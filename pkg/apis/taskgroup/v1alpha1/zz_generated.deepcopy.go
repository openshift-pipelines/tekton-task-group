//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ParamBinding) DeepCopyInto(out *ParamBinding) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ParamBinding.
func (in *ParamBinding) DeepCopy() *ParamBinding {
	if in == nil {
		return nil
	}
	out := new(ParamBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Step) DeepCopyInto(out *Step) {
	*out = *in
	in.Step.DeepCopyInto(&out.Step)
	if in.Uses != nil {
		in, out := &in.Uses, &out.Uses
		*out = new(Uses)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Step.
func (in *Step) DeepCopy() *Step {
	if in == nil {
		return nil
	}
	out := new(Step)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TaskGroup) DeepCopyInto(out *TaskGroup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TaskGroup.
func (in *TaskGroup) DeepCopy() *TaskGroup {
	if in == nil {
		return nil
	}
	out := new(TaskGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TaskGroup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TaskGroupList) DeepCopyInto(out *TaskGroupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TaskGroup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TaskGroupList.
func (in *TaskGroupList) DeepCopy() *TaskGroupList {
	if in == nil {
		return nil
	}
	out := new(TaskGroupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *TaskGroupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TaskGroupRunStatus) DeepCopyInto(out *TaskGroupRunStatus) {
	*out = *in
	if in.TaskGroupSpec != nil {
		in, out := &in.TaskGroupSpec, &out.TaskGroupSpec
		*out = new(TaskGroupSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.TaskRun != nil {
		in, out := &in.TaskRun, &out.TaskRun
		*out = new(v1beta1.TaskRunStatus)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TaskGroupRunStatus.
func (in *TaskGroupRunStatus) DeepCopy() *TaskGroupRunStatus {
	if in == nil {
		return nil
	}
	out := new(TaskGroupRunStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TaskGroupSpec) DeepCopyInto(out *TaskGroupSpec) {
	*out = *in
	if in.Params != nil {
		in, out := &in.Params, &out.Params
		*out = make([]v1beta1.ParamSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Steps != nil {
		in, out := &in.Steps, &out.Steps
		*out = make([]Step, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.StepTemplate != nil {
		in, out := &in.StepTemplate, &out.StepTemplate
		*out = new(v1.Container)
		(*in).DeepCopyInto(*out)
	}
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = make([]v1beta1.Sidecar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Workspaces != nil {
		in, out := &in.Workspaces, &out.Workspaces
		*out = make([]v1beta1.WorkspaceDeclaration, len(*in))
		copy(*out, *in)
	}
	if in.Results != nil {
		in, out := &in.Results, &out.Results
		*out = make([]v1beta1.TaskResult, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TaskGroupSpec.
func (in *TaskGroupSpec) DeepCopy() *TaskGroupSpec {
	if in == nil {
		return nil
	}
	out := new(TaskGroupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Uses) DeepCopyInto(out *Uses) {
	*out = *in
	in.TaskRef.DeepCopyInto(&out.TaskRef)
	if in.ParamBindings != nil {
		in, out := &in.ParamBindings, &out.ParamBindings
		*out = make([]ParamBinding, len(*in))
		copy(*out, *in)
	}
	if in.WorkspaceBindings != nil {
		in, out := &in.WorkspaceBindings, &out.WorkspaceBindings
		*out = make([]WorkspaceBinding, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Uses.
func (in *Uses) DeepCopy() *Uses {
	if in == nil {
		return nil
	}
	out := new(Uses)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkspaceBinding) DeepCopyInto(out *WorkspaceBinding) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkspaceBinding.
func (in *WorkspaceBinding) DeepCopy() *WorkspaceBinding {
	if in == nil {
		return nil
	}
	out := new(WorkspaceBinding)
	in.DeepCopyInto(out)
	return out
}
