package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/openshift-pipelines/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

func TestTaskGroupSpecValidate(t *testing.T) {
	tests := []struct {
		name string
		t    *v1alpha1.TaskGroupSpec
	}{{
		name: "uname steps",
		t: &v1alpha1.TaskGroupSpec{
			Steps: []v1alpha1.Step{{
				Uses: &v1alpha1.Uses{
					TaskRef: v1beta1.TaskRef{
						Bundle: "foo.bar/baz",
						Name:   "banana",
					},
				},
			}},
		},
	}}
	for _, tt := range tests {
		ctx := context.Background()
		tt.t.SetDefaults(ctx)
		if err := tt.t.Validate(ctx); err != nil {
			t.Errorf("TaskGroup.Validate() = %v", err)
		}
	}
}
func TestTaskGroupSpecValidateError(t *testing.T) {
	tests := []struct {
		name          string
		t             *v1alpha1.TaskGroupSpec
		expectedError apis.FieldError
	}{
		{
			name: "empty spec",
			t:    &v1alpha1.TaskGroupSpec{},
			expectedError: apis.FieldError{
				Message: `missing field(s)`,
				Paths:   []string{"steps"}},
		},
		// Uses
		{
			name: "empty uses",
			t: &v1alpha1.TaskGroupSpec{
				Steps: []v1alpha1.Step{{
					Uses: &v1alpha1.Uses{},
				}, {
					Step: v1beta1.Step{
						Container: corev1.Container{
							Image: "foo.bar/baz",
						},
					},
				}},
			},
			expectedError: apis.FieldError{
				Message: `missing field(s)`,
				Paths:   []string{"steps[0].uses.taskref.name"},
			},
		}}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tt.t.SetDefaults(ctx)
			err := tt.t.Validate(context.Background())
			if err == nil {
				t.Fatalf("Expected an error, got nothing for %v", tt.t)
			}
			t.Logf("%+v", err)
			if d := cmp.Diff(tt.expectedError.Error(), err.Error(), cmpopts.IgnoreUnexported(apis.FieldError{})); d != "" {
				t.Errorf("TaskSpec.Validate() errors diff %s", diff.PrintWantGot(d))
			}
		})
	}
}
