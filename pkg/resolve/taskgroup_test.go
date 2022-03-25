package resolve_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	// "github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
	"github.com/vdemeester/tekton-task-group/pkg/resolve"
)

func TestTaskSpec(t *testing.T) {
	cases := []struct {
		name          string
		taskGroupSpec *v1alpha1.TaskGroupSpec
		usedTaskSpec  map[int]v1beta1.TaskSpec
		expected      *v1beta1.TaskSpec
	}{{
		name: "no uses",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Steps: []v1alpha1.Step{{
				Step: v1beta1.Step{
					Container: corev1.Container{Image: "bash:latest"},
					Script:    "echo foo",
				},
			}},
		},
		usedTaskSpec: map[int]v1beta1.TaskSpec{},
		expected: &v1beta1.TaskSpec{
			Params: []v1beta1.ParamSpec{},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}},
		},
	}, {
		name: "uses steps",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Steps: []v1alpha1.Step{{
				Step: v1beta1.Step{
					Container: corev1.Container{Image: "bash:latest"},
					Script:    "echo foo",
				},
			}, {
				Step: v1beta1.Step{
					Container: corev1.Container{Name: "foo"},
				},
				Uses: &v1alpha1.Uses{
					TaskRef: v1beta1.TaskRef{Name: "foo"},
				},
			}},
		},
		usedTaskSpec: map[int]v1beta1.TaskSpec{
			1: v1beta1.TaskSpec{
				Steps: []v1beta1.Step{{
					Container: corev1.Container{Name: "baz", Image: "bash:latest"},
					Script:    "echo bar",
				}},
			},
		},
		expected: &v1beta1.TaskSpec{
			Params: []v1beta1.ParamSpec{},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}, {
				Container: corev1.Container{Name: "foo-baz", Image: "bash:latest"},
				Script:    "echo bar",
			}},
		},
	}}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			s, err := resolve.TaskSpec(tc.taskGroupSpec, tc.usedTaskSpec)
			if err != nil {
				t.Errorf("Failed to reslove taskspec: %v", err)
			}
			if d := cmp.Diff(tc.expected, s); d != "" {
				t.Errorf("Pod metadata doesn't match %s", diff.PrintWantGot(d))
			}
		})
	}
}

func TestTaskSpec_Invalid(t *testing.T) {
	cases := []struct {
		name          string
		taskGroupSpec *v1alpha1.TaskGroupSpec
		usedTaskSpec  map[int]v1beta1.TaskSpec
	}{{
		name: "no uses",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Steps: []v1alpha1.Step{{
				Uses: &v1alpha1.Uses{
					TaskRef: v1beta1.TaskRef{Name: "foo"},
				},
			}},
		},
		usedTaskSpec: map[int]v1beta1.TaskSpec{},
	}}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			s, err := resolve.TaskSpec(tc.taskGroupSpec, tc.usedTaskSpec)
			if err == nil {
				t.Errorf("Should have failed, did not : %+v", s)
			}
		})
	}
}
