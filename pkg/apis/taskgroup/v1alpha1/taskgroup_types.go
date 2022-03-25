/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskGroup iteratively executes a Task over elements in an array.
// +k8s:openapi-gen=true
type TaskGroup struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the TaskGroup from the client
	// +optional
	Spec TaskGroupSpec `json:"spec"`
}

// TaskGroupSpec defines the desired state of the TaskGroup
type TaskGroupSpec struct {
	// FIXME(vdemeester): define a spec

	// Params is a list of input parameters required to run the task. Params
	// must be supplied as inputs in TaskRuns unless they declare a default
	// value.
	// +optional
	Params []v1beta1.ParamSpec `json:"params,omitempty"`

	// Description is a user-facing description of the task that may be
	// used to populate a UI.
	// +optional
	Description string `json:"description,omitempty"`

	// Steps are the steps of the build; each step is run sequentially with the
	// source mounted into /workspace.
	Steps []Step `json:"steps,omitempty"`

	// Volumes is a collection of volumes that are available to mount into the
	// steps of the build.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// StepTemplate can be used as the basis for all step containers within the
	// Task, so that the steps inherit settings on the base container.
	StepTemplate *corev1.Container `json:"stepTemplate,omitempty"`

	// Sidecars are run alongside the Task's step containers. They begin before
	// the steps start and end after the steps complete.
	Sidecars []v1beta1.Sidecar `json:"sidecars,omitempty"`

	// Workspaces are the volumes that this Task requires.
	Workspaces []v1beta1.WorkspaceDeclaration `json:"workspaces,omitempty"`

	// Results are values that this Task can output
	Results []v1beta1.TaskResult `json:"results,omitempty"`
}

type Step struct {
	v1beta1.Step `json:",inline"`

	// +optional
	Uses *Uses `json:"uses"`
}

type Uses struct {
	TaskRef v1beta1.TaskRef `json:"taskRef"`

	ParamBindings     []ParamBinding     `json:"parambindings"`
	WorkspaceBindings []WorkspaceBinding `json:"workspacebinding"`
}

type ParamBinding struct {
	Name  string `json:"name"`
	Param string `json:"param"`
}

type WorkspaceBinding struct {
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskGroupList contains a list of TaskGroups
type TaskGroupList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskGroup `json:"items"`
}

// TaskGroupRunReason represents a reason for the Run "Succeeded" condition
type TaskGroupRunReason string

const (
	// TaskGroupRunReasonStarted is the reason set when the Run has just started
	TaskGroupRunReasonStarted TaskGroupRunReason = "Started"

	// TaskGroupRunReasonRunning indicates that the Run is in progress
	TaskGroupRunReasonRunning TaskGroupRunReason = "Running"

	// TaskGroupRunReasonFailed indicates that one of the TaskRuns created from the Run failed
	TaskGroupRunReasonFailed TaskGroupRunReason = "Failed"

	// TaskGroupRunReasonSucceeded indicates that all of the TaskRuns created from the Run completed successfully
	TaskGroupRunReasonSucceeded TaskGroupRunReason = "Succeeded"

	// TaskGroupRunReasonCouldntCancel indicates that a Run was cancelled but attempting to update
	// the running TaskRun as cancelled failed.
	TaskGroupRunReasonCouldntCancel TaskGroupRunReason = "TaskGroupRunCouldntCancel"

	// TaskGroupRunReasonCouldntGetTaskGroup indicates that the associated TaskGroup couldn't be retrieved
	TaskGroupRunReasonCouldntGetTaskGroup TaskGroupRunReason = "CouldntGetTaskGroup"

	// TaskGroupRunReasonFailedValidation indicates that the TaskGroup failed runtime validation
	TaskGroupRunReasonFailedValidation TaskGroupRunReason = "TaskGroupValidationFailed"

	// TaskGroupRunReasonInternalError indicates that the TaskGroup failed due to an internal error in the reconciler
	TaskGroupRunReasonInternalError TaskGroupRunReason = "TaskGroupInternalError"
)

func (t TaskGroupRunReason) String() string {
	return string(t)
}

// TaskGroupRunStatus contains the status stored in the ExtraFields of a Run that references a TaskGroup.
type TaskGroupRunStatus struct {
	// TaskGroupSpec contains the exact spec used to instantiate the Run
	// FIXME(vdemeester) can probably remove
	TaskGroupSpec *TaskGroupSpec `json:"taskLoopSpec,omitempty"`
	// +optional
	TaskRun *v1beta1.TaskRunStatus `json:"status,omitempty"`
}
