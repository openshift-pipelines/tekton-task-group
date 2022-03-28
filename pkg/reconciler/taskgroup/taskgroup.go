package taskgroup

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	listersalpha "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/names"
	"github.com/tektoncd/pipeline/pkg/reconciler/events"
	"github.com/tektoncd/pipeline/pkg/reconciler/taskrun/resources"
	"github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup"
	taskgroupv1alpha1 "github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
	taskgroupclientset "github.com/vdemeester/tekton-task-group/pkg/client/clientset/versioned"
	listerstaskgroup "github.com/vdemeester/tekton-task-group/pkg/client/listers/taskgroup/v1alpha1"
	"github.com/vdemeester/tekton-task-group/pkg/resolve"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// taskGroupLabelKey is the label identifier for a TaskGroup.  This label is added to the Run and its TaskRuns.
	taskGroupLabelKey = "/taskGroup"

	// taskGroupRunLabelKey is the label identifier for a Run.  This label is added to the Run's TaskRuns.
	taskGroupRunLabelKey = "/run"
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	pipelineClientSet  clientset.Interface
	kubeClientSet      kubernetes.Interface
	taskgroupClientSet taskgroupclientset.Interface
	runLister          listersalpha.RunLister
	taskGroupLister    listerstaskgroup.TaskGroupLister
	taskRunLister      listers.TaskRunLister
}

var (
	// Check that our Reconciler implements runreconciler.Interface
	_                runreconciler.Interface = (*Reconciler)(nil)
	cancelPatchBytes []byte
)

func init() {
	var err error
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: "add",
		Path:      "/spec/status",
		Value:     v1beta1.TaskRunSpecStatusCancelled,
	}}
	cancelPatchBytes, err = json.Marshal(patches)
	if err != nil {
		log.Fatalf("failed to marshal patch bytes in order to cancel: %v", err)
	}
}

// ReconcileKind compares the actual state with the desired, and attempts to converge the two.
// It then updates the Status block of the Run resource with the current status of the resource.
func (c *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) pkgreconciler.Event {
	var merr error
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling Run %s/%s at %v", run.Namespace, run.Name, time.Now())

	// Check that the Run references a TaskGroup CRD.  The logic is controller.go should ensure that only this type of Run
	// is reconciled this controller but it never hurts to do some bullet-proofing.
	var apiVersion, kind string
	if run.Spec.Ref != nil {
		apiVersion = run.Spec.Ref.APIVersion
		kind = string(run.Spec.Ref.Kind)
	} else if run.Spec.Spec != nil {
		apiVersion = run.Spec.Spec.APIVersion
		kind = run.Spec.Spec.Kind
	}
	if apiVersion != taskgroupv1alpha1.SchemeGroupVersion.String() ||
		kind != taskgroup.TaskGroupControllerName {
		logger.Errorf("Received control for a Run %s/%s that does not reference a TaskGroup custom CRD", run.Namespace, run.Name)
		return nil
	}

	// If the Run has not started, initialize the Condition and set the start time.
	if !run.HasStarted() {
		logger.Infof("Starting new Run %s/%s", run.Namespace, run.Name)
		run.Status.InitializeConditions()
		// In case node time was not synchronized, when controller has been scheduled to other nodes.
		if run.Status.StartTime.Sub(run.CreationTimestamp.Time) < 0 {
			logger.Warnf("Run %s createTimestamp %s is after the Run started %s", run.Name, run.CreationTimestamp, run.Status.StartTime)
			run.Status.StartTime = &run.CreationTimestamp
		}
		// Emit events. During the first reconcile the status of the Run may change twice
		// from not Started to Started and then to Running, so we need to sent the event here
		// and at the end of 'Reconcile' again.
		// We also want to send the "Started" event as soon as possible for anyone who may be waiting
		// on the event to perform user facing initialisations, such has reset a CI check status
		afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
		events.Emit(ctx, nil, afterCondition, run)
	}

	if run.IsDone() {
		logger.Infof("Run %s/%s is done", run.Namespace, run.Name)
		return nil
	}

	// Store the condition before reconcile
	beforeCondition := run.Status.GetCondition(apis.ConditionSucceeded)

	status := &taskgroupv1alpha1.TaskGroupRunStatus{}
	if err := run.Status.DecodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(taskgroupv1alpha1.TaskGroupRunReasonInternalError.String(),
			"Internal error calling DecodeExtraFields: %v", err)
		logger.Errorf("DecodeExtraFields error: %v", err.Error())
	}

	// Reconcile the Run
	if err := c.reconcile(ctx, run, status); err != nil {
		logger.Errorf("Reconcile error: %v", err.Error())
		merr = multierror.Append(merr, err)
	}

	if err := c.updateLabelsAndAnnotations(ctx, run); err != nil {
		logger.Warn("Failed to update Run labels/annotations", zap.Error(err))
		merr = multierror.Append(merr, err)
	}

	if err := run.Status.EncodeExtraFields(status); err != nil {
		run.Status.MarkRunFailed(taskgroupv1alpha1.TaskGroupRunReasonInternalError.String(),
			"Internal error calling EncodeExtraFields: %v", err)
		logger.Errorf("EncodeExtraFields error: %v", err.Error())
	}

	afterCondition := run.Status.GetCondition(apis.ConditionSucceeded)
	events.Emit(ctx, beforeCondition, afterCondition, run)

	// Only transient errors that should retry the reconcile are returned.
	return merr
}

func (c *Reconciler) reconcile(ctx context.Context, run *v1alpha1.Run, status *taskgroupv1alpha1.TaskGroupRunStatus) error {
	logger := logging.FromContext(ctx)

	// Get the TaskGroup referenced by the Run
	taskGroupMeta, taskGroupSpec, err := c.getTaskGroup(ctx, logger, run)
	if err != nil {
		return err
	}

	// Store the fetched TaskGroupSpec on the Run for auditing
	storeTaskGroupSpec(status, taskGroupSpec)

	// Propagate labels and annotations from TaskGroup to Run.
	propagateTaskGroupLabelsAndAnnotations(run, taskGroupMeta)

	// Validate TaskGroup spec
	if err := taskGroupSpec.Validate(ctx); err != nil {
		run.Status.MarkRunFailed(taskgroupv1alpha1.TaskGroupRunReasonFailedValidation.String(),
			"TaskGroup %s/%s can't be Run; it has an invalid spec: %s",
			taskGroupMeta.Namespace, taskGroupMeta.Name, err)
		return nil
	}

	taskRunDone, taskRunFailed, err := c.updateTaskRunStatus(ctx, logger, run, status, taskGroupSpec)
	if err != nil {
		return fmt.Errorf("error updating TaskRun status for Run %s/%s: %w", run.Namespace, run.Name, err)
	}
	if taskRunDone {
		if taskRunFailed {
			run.Status.MarkRunFailed(taskgroupv1alpha1.TaskGroupRunReasonFailed.String(),
				"TaskRun failed")
			return nil
		} else {
			run.Status.MarkRunSucceeded(taskgroupv1alpha1.TaskGroupRunReasonSucceeded.String(),
				"TaskRun succeeded")
			return nil
		}
	} else {
		run.Status.MarkRunRunning(taskgroupv1alpha1.TaskGroupRunReasonRunning.String(),
			"%s running", run.Name)
	}

	if status.TaskRun == nil {
		// Create a TaskRun, with embedded spec based on multiple Tasks
		tr, err := c.createTaskRun(ctx, logger, taskGroupSpec, run)
		if err != nil {
			run.Status.MarkRunFailed(taskgroupv1alpha1.TaskGroupRunReasonFailedValidation.String(),
				"TaskGroup %s/%s can't be Run; failed to create TaskRun: %s",
				taskGroupMeta.Namespace, taskGroupMeta.Name, err)
			return nil
		}
		status.TaskRun = &tr.Status
	}

	return nil
}

func (c *Reconciler) getTaskGroup(ctx context.Context, logger *zap.SugaredLogger, run *v1alpha1.Run) (*metav1.ObjectMeta, *taskgroupv1alpha1.TaskGroupSpec, error) {
	taskGroupMeta := metav1.ObjectMeta{}
	taskGroupSpec := taskgroupv1alpha1.TaskGroupSpec{}
	if run.Spec.Ref != nil && run.Spec.Ref.Name != "" {
		// Use the k8 client to get the TaskGroup rather than the lister.  This avoids a timing issue where
		// the TaskGroup is not yet in the lister cache if it is created at nearly the same time as the Run.
		// See https://github.com/tektoncd/pipeline/issues/2740 for discussion on this issue.
		//
		tl, err := c.taskgroupClientSet.CustomV1alpha1().TaskGroups(run.Namespace).Get(ctx, run.Spec.Ref.Name, metav1.GetOptions{})
		if err != nil {
			run.Status.MarkRunFailed(taskgroupv1alpha1.TaskGroupRunReasonCouldntGetTaskGroup.String(),
				"Error retrieving TaskGroup for Run %s/%s: %s",
				run.Namespace, run.Name, err)
			return nil, nil, fmt.Errorf("Error retrieving TaskGroup for Run %s: %w", fmt.Sprintf("%s/%s", run.Namespace, run.Name), err)
		}
		taskGroupMeta = tl.ObjectMeta
		taskGroupSpec = tl.Spec
	} else if run.Spec.Spec != nil {
		// FIXME(vdemeester) support embedded spec
		if err := json.Unmarshal(run.Spec.Spec.Spec.Raw, &taskGroupSpec); err != nil {
			run.Status.MarkRunFailed(taskgroupv1alpha1.TaskGroupRunReasonCouldntGetTaskGroup.String(),
				"Error retrieving TaskGroup for Run %s/%s: %s",
				run.Namespace, run.Name, err)
			return nil, nil, fmt.Errorf("Error retrieving TaskGroup for Run %s: %w", fmt.Sprintf("%s/%s", run.Namespace, run.Name), err)
		}
	}
	return &taskGroupMeta, &taskGroupSpec, nil
}

func (c *Reconciler) createTaskRun(ctx context.Context, logger *zap.SugaredLogger, spec *taskgroupv1alpha1.TaskGroupSpec, run *v1alpha1.Run) (*v1beta1.TaskRun, error) {
	usedTaskSpecs, err := c.fetchUsesTaskSpec(ctx, logger, spec, run)
	if err != nil {
		return nil, err
	}
	taskSpec, err := resolve.TaskSpec(spec, usedTaskSpecs)
	if err != nil {
		return nil, err
	}

	// Create name for TaskRun from Run name plus iteration number.
	trName := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix(run.Name)

	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            trName,
			Namespace:       run.Namespace,
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(run)},
			Labels:          getTaskRunLabels(run, true),
			Annotations:     getTaskRunAnnotations(run),
		},
		Spec: v1beta1.TaskRunSpec{
			Params:             run.Spec.Params,
			ServiceAccountName: run.Spec.ServiceAccountName,
			PodTemplate:        run.Spec.PodTemplate,
			Workspaces:         run.Spec.Workspaces,
			TaskSpec:           taskSpec,
		},
	}

	logger.Infof("Creating a new TaskRun object %s", trName)
	return c.pipelineClientSet.TektonV1beta1().TaskRuns(run.Namespace).Create(ctx, tr, metav1.CreateOptions{})
}

func (c *Reconciler) fetchUsesTaskSpec(ctx context.Context, logger *zap.SugaredLogger, spec *taskgroupv1alpha1.TaskGroupSpec, run *v1alpha1.Run) (map[int]v1beta1.TaskSpec, error) {
	taskSpecs := map[int]v1beta1.TaskSpec{}
	for i, step := range spec.Steps {
		if step.Uses == nil {
			continue
		}
		getTaskfunc, err := resources.GetTaskFunc(ctx, c.kubeClientSet, c.pipelineClientSet, &step.Uses.TaskRef, run.Namespace, run.Spec.ServiceAccountName)
		if err != nil {
			logger.Errorf("Failed to fetch task reference %s: %v", step.Uses.TaskRef.Name, err)
			return nil, errors.Wrapf(err, "failed to fetch task reference %s", step.Uses.TaskRef.Name)
		}

		_, taskSpec, err := resources.GetTaskData(ctx, &v1beta1.TaskRun{
			Spec: v1beta1.TaskRunSpec{TaskRef: &step.Uses.TaskRef},
		}, getTaskfunc)
		if err != nil {
			logger.Errorf("Failed to fetch task reference %s: %v", step.Uses.TaskRef.Name, err)
			if resources.IsGetTaskErrTransient(err) {
				return nil, err
			}
			return nil, controller.NewPermanentError(err)
		}
		taskSpecs[i] = *taskSpec

	}
	return taskSpecs, nil
}

func (c *Reconciler) updateLabelsAndAnnotations(ctx context.Context, run *v1alpha1.Run) error {
	newRun, err := c.runLister.Runs(run.Namespace).Get(run.Name)
	if err != nil {
		return fmt.Errorf("error getting Run %s when updating labels/annotations: %w", run.Name, err)
	}
	if !reflect.DeepEqual(run.ObjectMeta.Labels, newRun.ObjectMeta.Labels) || !reflect.DeepEqual(run.ObjectMeta.Annotations, newRun.ObjectMeta.Annotations) {
		mergePatch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels":      run.ObjectMeta.Labels,
				"annotations": run.ObjectMeta.Annotations,
			},
		}
		patch, err := json.Marshal(mergePatch)
		if err != nil {
			return err
		}
		_, err = c.pipelineClientSet.TektonV1alpha1().Runs(run.Namespace).Patch(ctx, run.Name, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	}
	return nil
}

func (c *Reconciler) updateTaskRunStatus(ctx context.Context, logger *zap.SugaredLogger, run *v1alpha1.Run, status *taskgroupv1alpha1.TaskGroupRunStatus,
	taskGroupSpec *taskgroupv1alpha1.TaskGroupSpec) (taskRunDone bool, taskRunFailed bool, retryableErr error) {
	// List TaskRuns associated with this Run.  These TaskRuns should be recorded in the Run status but it's
	// possible that this reconcile call has been passed stale status which doesn't include a previous update.
	// Find the TaskRuns by matching labels.  Do not include the propagated labels from the Run.
	// The user could change them during the lifetime of the Run so the current labels may not be set on the
	// previously created TaskRuns.
	taskRunLabels := getTaskRunLabels(run, false)
	taskRuns, err := c.taskRunLister.TaskRuns(run.Namespace).List(labels.SelectorFromSet(taskRunLabels))
	if err != nil {
		retryableErr = fmt.Errorf("could not list TaskRuns %#v", err)
		return
	}
	if taskRuns == nil || len(taskRuns) == 0 {
		return
	}
	if len(taskRuns) > 1 {
		retryableErr = fmt.Errorf("Multiple taskRuns, this is a problem")
		return
	}
	tr := taskRuns[0]

	// lbls := tr.GetLabels()
	status.TaskRun = &tr.Status
	// If the TaskRun was created before the Run says it was started, then change the Run's
	// start time.  This happens when this reconcile call has been passed stale status that
	// doesn't have the start time set.  The reconcile call will set a new start time that
	// is later than TaskRuns it previously created.  The Run start time is adjusted back
	// to compensate for this problem.
	if tr.CreationTimestamp.Before(run.Status.CompletionTime) {
		run.Status.CompletionTime = tr.CreationTimestamp.DeepCopy()
	}
	// Handle TaskRun cancellation and retry.
	if err := c.processTaskRun(ctx, logger, tr, run, status, taskGroupSpec); err != nil {
		retryableErr = fmt.Errorf("error processing TaskRun %s: %#v", tr.Name, err)
		return
	}
	if tr.IsDone() {
		taskRunDone = true
		if !tr.IsSuccessful() {
			taskRunFailed = true
		}
	}

	return
}

func (c *Reconciler) processTaskRun(ctx context.Context, logger *zap.SugaredLogger, tr *v1beta1.TaskRun,
	run *v1alpha1.Run, status *taskgroupv1alpha1.TaskGroupRunStatus, taskGroupSpec *taskgroupv1alpha1.TaskGroupSpec) error {
	// If the TaskRun is running and the Run is cancelled, cancel the TaskRun.
	if !tr.IsDone() {
		if run.IsCancelled() && !tr.IsCancelled() {
			logger.Infof("Run %s/%s is cancelled.  Cancelling TaskRun %s.", run.Namespace, run.Name, tr.Name)
			if _, err := c.pipelineClientSet.TektonV1beta1().TaskRuns(run.Namespace).Patch(ctx, tr.Name, types.JSONPatchType, cancelPatchBytes, metav1.PatchOptions{}); err != nil {
				return fmt.Errorf("Failed to patch TaskRun `%s` with cancellation: %v", tr.Name, err)
			}
		}
	}
	return nil
}

func getTaskRunAnnotations(run *v1alpha1.Run) map[string]string {
	// Propagate annotations from Run to TaskRun.
	annotations := make(map[string]string, len(run.ObjectMeta.Annotations)+1)
	for key, val := range run.ObjectMeta.Annotations {
		annotations[key] = val
	}
	return annotations
}

func getTaskRunLabels(run *v1alpha1.Run, includeRunLabels bool) map[string]string {
	// Propagate labels from Run to TaskRun.
	labels := make(map[string]string, len(run.ObjectMeta.Labels)+1)
	if includeRunLabels {
		for key, val := range run.ObjectMeta.Labels {
			labels[key] = val
		}
	}
	// Note: The Run label uses the normal Tekton group name.
	labels[pipeline.GroupName+taskGroupRunLabelKey] = run.Name
	return labels
}

func propagateTaskGroupLabelsAndAnnotations(run *v1alpha1.Run, taskGroupMeta *metav1.ObjectMeta) {
	// Propagate labels from TaskGroup to Run.
	if run.ObjectMeta.Labels == nil {
		run.ObjectMeta.Labels = make(map[string]string, len(taskGroupMeta.Labels)+1)
	}
	for key, value := range taskGroupMeta.Labels {
		run.ObjectMeta.Labels[key] = value
	}
	run.ObjectMeta.Labels[taskgroup.GroupName+taskGroupLabelKey] = taskGroupMeta.Name

	// Propagate annotations from TaskGroup to Run.
	if run.ObjectMeta.Annotations == nil {
		run.ObjectMeta.Annotations = make(map[string]string, len(taskGroupMeta.Annotations))
	}
	for key, value := range taskGroupMeta.Annotations {
		run.ObjectMeta.Annotations[key] = value
	}
}

func storeTaskGroupSpec(status *taskgroupv1alpha1.TaskGroupRunStatus, tls *taskgroupv1alpha1.TaskGroupSpec) {
	// Only store the TaskGroupSpec once, if it has never been set before.
	if status.TaskGroupSpec == nil {
		status.TaskGroupSpec = tls
	}
}
