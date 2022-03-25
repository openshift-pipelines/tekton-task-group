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
			Results:    []v1beta1.TaskResult{},
			Params:     []v1beta1.ParamSpec{},
			Workspaces: []v1beta1.WorkspaceDeclaration{},
			Sidecars:   []v1beta1.Sidecar{},
			Volumes:    []corev1.Volume{},
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
			Sidecars:   []v1beta1.Sidecar{},
			Volumes:    []corev1.Volume{},
			Results:    []v1beta1.TaskResult{},
			Params:     []v1beta1.ParamSpec{},
			Workspaces: []v1beta1.WorkspaceDeclaration{},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}, {
				Container: corev1.Container{Name: "foo-baz", Image: "bash:latest"},
				Script:    "echo bar",
			}},
		},
	}, {
		name: "uses steps and params",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Params: []v1beta1.ParamSpec{{
				Name: "paramTaskGroup",
				Type: v1beta1.ParamTypeString,
			}},
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
				Params: []v1beta1.ParamSpec{{
					Name:        "param1",
					Type:        v1beta1.ParamTypeString,
					Default:     v1beta1.NewArrayOrString("value1"),
					Description: "description1",
				}},
				Steps: []v1beta1.Step{{
					Container: corev1.Container{Name: "baz", Image: "bash:latest"},
					Script:    "echo bar",
				}},
			},
		},
		expected: &v1beta1.TaskSpec{
			Sidecars:   []v1beta1.Sidecar{},
			Volumes:    []corev1.Volume{},
			Results:    []v1beta1.TaskResult{},
			Workspaces: []v1beta1.WorkspaceDeclaration{},
			Params: []v1beta1.ParamSpec{{
				Name: "paramTaskGroup",
				Type: v1beta1.ParamTypeString,
			}, {
				Name:        "param1",
				Type:        v1beta1.ParamTypeString,
				Default:     v1beta1.NewArrayOrString("value1"),
				Description: "description1",
			}},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}, {
				Container: corev1.Container{Name: "foo-baz", Image: "bash:latest"},
				Script:    "echo bar",
			}},
		},
	}, {
		name: "uses steps and duplicated params",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Params: []v1beta1.ParamSpec{{
				Name: "paramTaskGroup",
				Type: v1beta1.ParamTypeString,
			}, {
				Name: "paramFoo",
				Type: v1beta1.ParamTypeString,
			}},
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
				Params: []v1beta1.ParamSpec{{
					Name: "paramFoo",
					Type: v1beta1.ParamTypeString,
				}},
				Steps: []v1beta1.Step{{
					Container: corev1.Container{Name: "baz", Image: "bash:latest"},
					Script:    "echo bar",
				}},
			},
		},
		expected: &v1beta1.TaskSpec{
			Sidecars:   []v1beta1.Sidecar{},
			Volumes:    []corev1.Volume{},
			Results:    []v1beta1.TaskResult{},
			Workspaces: []v1beta1.WorkspaceDeclaration{},
			Params: []v1beta1.ParamSpec{{
				Name: "paramTaskGroup",
				Type: v1beta1.ParamTypeString,
			}, {
				Name: "paramFoo",
				Type: v1beta1.ParamTypeString,
			}},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}, {
				Container: corev1.Container{Name: "foo-baz", Image: "bash:latest"},
				Script:    "echo bar",
			}},
		},
	}, {
		name: "uses steps and param bindings",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Params: []v1beta1.ParamSpec{{
				Name: "paramFoo",
				Type: v1beta1.ParamTypeString,
			}},
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
					ParamBindings: []v1alpha1.ParamBinding{{
						Name:  "paramBar",
						Param: "paramFoo",
					}},
				},
			}},
		},
		usedTaskSpec: map[int]v1beta1.TaskSpec{
			1: v1beta1.TaskSpec{
				Params: []v1beta1.ParamSpec{{
					Name: "paramBar",
					Type: v1beta1.ParamTypeString,
				}},
				Steps: []v1beta1.Step{{
					Container: corev1.Container{Name: "baz", Image: "bash:latest"},
					Script:    "echo $(params.paramBar) $(params['paramBar']) $(params[\"paramBar\"])",
				}},
			},
		},
		expected: &v1beta1.TaskSpec{
			Sidecars:   []v1beta1.Sidecar{},
			Volumes:    []corev1.Volume{},
			Results:    []v1beta1.TaskResult{},
			Workspaces: []v1beta1.WorkspaceDeclaration{},
			Params: []v1beta1.ParamSpec{{
				Name: "paramFoo",
				Type: v1beta1.ParamTypeString,
			}},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}, {
				Container: corev1.Container{Name: "foo-baz", Image: "bash:latest"},
				Script:    "echo $(params.paramFoo) $(params['paramFoo']) $(params[\"paramFoo\"])",
			}},
		},
	}, {
		name: "uses steps and workspace bindings",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name: "ws1",
			}},
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
					WorkspaceBindings: []v1alpha1.WorkspaceBinding{{
						Name:      "myws",
						Workspace: "ws1",
					}},
				},
			}},
		},
		usedTaskSpec: map[int]v1beta1.TaskSpec{
			1: v1beta1.TaskSpec{
				Workspaces: []v1beta1.WorkspaceDeclaration{{
					Name: "myws",
				}},
				Steps: []v1beta1.Step{{
					Container: corev1.Container{Name: "baz", Image: "bash:latest"},
					Script:    "echo $(workspaces.myws.path) $(workspaces.myws.bound) $(workspaces.myws.claim) $(workspaces.myws.volume)",
				}},
			},
		},
		expected: &v1beta1.TaskSpec{
			Sidecars: []v1beta1.Sidecar{},
			Volumes:  []corev1.Volume{},
			Results:  []v1beta1.TaskResult{},
			Params:   []v1beta1.ParamSpec{},
			Workspaces: []v1beta1.WorkspaceDeclaration{{
				Name: "ws1",
			}},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}, {
				Container: corev1.Container{Name: "foo-baz", Image: "bash:latest"},
				Script:    "echo $(workspaces.ws1.path) $(workspaces.ws1.bound) $(workspaces.ws1.claim) $(workspaces.ws1.volume)",
			}},
		},
	}, {
		name: "uses steps and results",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Results: []v1beta1.TaskResult{{
				Name: "foo",
			}},
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
				Results: []v1beta1.TaskResult{{
					Name: "foo",
				}, {
					Name: "bar",
				}},
				Steps: []v1beta1.Step{{
					Container: corev1.Container{Name: "baz", Image: "bash:latest"},
					Script:    "echo bar",
				}},
			},
		},
		expected: &v1beta1.TaskSpec{
			Sidecars: []v1beta1.Sidecar{},
			Volumes:  []corev1.Volume{},
			Results: []v1beta1.TaskResult{{
				Name: "foo",
			}, {
				Name: "bar",
			}},
			Workspaces: []v1beta1.WorkspaceDeclaration{},
			Params:     []v1beta1.ParamSpec{},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}, {
				Container: corev1.Container{Name: "foo-baz", Image: "bash:latest"},
				Script:    "echo bar",
			}},
		},
	}, {
		name: "uses steps and stepTemplate",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			StepTemplate: &corev1.Container{
				Image: "bash:latest",
			},
			Steps: []v1alpha1.Step{{
				Step: v1beta1.Step{
					Script: "echo foo",
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
				StepTemplate: &corev1.Container{
					Image: "fedora:latest",
				},
				Steps: []v1beta1.Step{{
					Container: corev1.Container{Name: "bar"},
					Script:    "echo bar",
				}},
			},
		},
		expected: &v1beta1.TaskSpec{
			Sidecars:   []v1beta1.Sidecar{},
			Volumes:    []corev1.Volume{},
			Results:    []v1beta1.TaskResult{},
			Workspaces: []v1beta1.WorkspaceDeclaration{},
			Params:     []v1beta1.ParamSpec{},
			Steps: []v1beta1.Step{{
				Container: corev1.Container{Image: "bash:latest"},
				Script:    "echo foo",
			}, {
				Container: corev1.Container{Name: "foo-bar", Image: "fedora:latest"},
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
		name: "uses but no use picked up (not a valid case in prod)",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Steps: []v1alpha1.Step{{
				Uses: &v1alpha1.Uses{
					TaskRef: v1beta1.TaskRef{Name: "foo"},
				},
			}},
		},
		usedTaskSpec: map[int]v1beta1.TaskSpec{},
	}, {
		name: "duplicated params with different types",
		taskGroupSpec: &v1alpha1.TaskGroupSpec{
			Params: []v1beta1.ParamSpec{{
				Name: "paramFoo",
				Type: v1beta1.ParamTypeString,
			}},
			Steps: []v1alpha1.Step{{
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
				Params: []v1beta1.ParamSpec{{
					Name: "paramFoo",
					Type: v1beta1.ParamTypeArray,
				}},
				Steps: []v1beta1.Step{{
					Container: corev1.Container{Name: "baz", Image: "bash:latest"},
					Script:    "echo bar",
				}},
			},
		},
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
