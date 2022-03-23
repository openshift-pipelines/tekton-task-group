package taskgroup

import (
	"context"

	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	pipelinecontroller "github.com/tektoncd/pipeline/pkg/controller"
	"github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup"
	taskgroupv1alpha1 "github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
	taskgroupclient "github.com/vdemeester/tekton-task-group/pkg/client/injection/client"
	taskgroupinformer "github.com/vdemeester/tekton-task-group/pkg/client/injection/informers/taskgroup/v1alpha1/taskgroup"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController() func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {

		logger := logging.FromContext(ctx)
		pipelineclientset := pipelineclient.Get(ctx)
		taskgroupclientset := taskgroupclient.Get(ctx)
		runInformer := runinformer.Get(ctx)
		taskGroupInformer := taskgroupinformer.Get(ctx)
		taskRunInformer := taskruninformer.Get(ctx)

		c := &Reconciler{
			pipelineClientSet:  pipelineclientset,
			taskgroupClientSet: taskgroupclientset,
			runLister:          runInformer.Lister(),
			taskGroupLister:    taskGroupInformer.Lister(),
			taskRunLister:      taskRunInformer.Lister(),
		}

		impl := runreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: "run-taskgroup",
			}
		})

		logger.Info("Setting up event handlers")

		// Add event handler for Runs
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterRunRef(taskgroupv1alpha1.SchemeGroupVersion.String(), taskgroup.TaskGroupControllerName),
			Handler:    controller.HandleAll(impl.Enqueue),
		})

		// Add event handler for TaskRuns controlled by Run
		taskRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: pipelinecontroller.FilterOwnerRunRef(runInformer.Lister(), taskgroupv1alpha1.SchemeGroupVersion.String(), taskgroup.TaskGroupControllerName),
			Handler:    controller.HandleAll(impl.EnqueueControllerOf),
		})

		return impl
	}
}
