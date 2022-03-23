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
	Steps []Step `json:"steps"`
}

type Step struct {
	v1beta1.Step `json:",inline"`

	// +optional
	Uses *Uses `json:"uses"`
}

type Uses struct {
	TaskRef v1beta1.TaskRef `json:"taskRef"`
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
	TaskGroupSpec *TaskGroupSpec `json:"taskLoopSpec,omitempty"`
	// map of TaskGroupTaskRunStatus with the taskRun name as the key
	// +optional
	TaskRuns map[string]*TaskGroupTaskRunStatus `json:"taskRuns,omitempty"`
}

// TaskGroupTaskRunStatus contains the iteration number for a TaskRun and the TaskRun's Status
type TaskGroupTaskRunStatus struct {
	// iteration number
	Iteration int `json:"iteration,omitempty"`
	// Status is the TaskRunStatus for the corresponding TaskRun
	// +optional
	Status *v1beta1.TaskRunStatus `json:"status,omitempty"`
}
