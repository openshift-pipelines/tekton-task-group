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
	"context"
	// "fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/validate"
	corev1 "k8s.io/api/core/v1"
	// "k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*TaskGroup)(nil)

// Validate TaskGroup
func (tg *TaskGroup) Validate(ctx context.Context) *apis.FieldError {
	if err := validate.ObjectMetadata(tg.GetObjectMeta()); err != nil {
		return err.ViaField("metadata")
	}
	return tg.Spec.Validate(ctx)
}

// Validate TaskGroupSpec
func (tgs *TaskGroupSpec) Validate(ctx context.Context) (errs *apis.FieldError) {
	// Try to share as much validation with upstream
	if len(tgs.Steps) == 0 {
		errs = errs.Also(apis.ErrMissingField("steps"))
	}
	steps := toUpstreamStep(tgs.Steps)
	errs = errs.Also(validateSteps(ctx, tgs.Steps, steps, tgs.StepTemplate).ViaField("steps"))
	errs = errs.Also(v1beta1.ValidateVolumes(tgs.Volumes).ViaField("volumes"))
	// errs = errs.Also(v1beta1.ValidateDeclaredWorkspaces(tgs.Workspaces, steps, tgs.StepTemplate).ViaField("workspaces"))
	// errs = errs.Also(v1beta1.ValidateWorkspaceUsages(ctx, tgs))
	errs = errs.Also(v1beta1.ValidateParameterTypes(tgs.Params).ViaField("params"))
	errs = errs.Also(v1beta1.ValidateParameterVariables(steps, tgs.Params))
	errs = errs.Also(v1beta1.ValidateResourcesVariables(steps, nil))
	// errs = errs.Also(v1beta1.ValidateTaskContextVariables(steps))
	errs = errs.Also(validateResults(ctx, tgs.Results).ViaField("results"))
	return errs
}

func validateResults(ctx context.Context, results []v1beta1.TaskResult) (errs *apis.FieldError) {
	for index, result := range results {
		errs = errs.Also(result.Validate(ctx).ViaIndex(index))
	}
	return errs
}

func validateSteps(ctx context.Context, steps []Step, usteps []v1beta1.Step, stepTemplate *corev1.Container) (errs *apis.FieldError) {
	// mergedSteps, err := v1beta1.MergeStepsWithStepTemplate(stepTemplate, usteps)
	// if err != nil {
	// 	errs = errs.Also(&apis.FieldError{
	// 		Message: fmt.Sprintf("error merging step template and steps: %s", err),
	// 		Paths:   []string{"stepTemplate"},
	// 		Details: err.Error(),
	// 	})
	// }
	// names := sets.NewString()
	for idx, s := range steps {
		// errs = errs.Also(mergedSteps[idx].Validate(ctx, names).ViaIndex(idx))
		errs = errs.Also(s.Validate(ctx).ViaIndex(idx))
	}
	return errs
}

func (s Step) Validate(ctx context.Context) (errs *apis.FieldError) {
	errs = errs.Also(s.Uses.Validate(ctx).ViaField("uses"))
	return errs
}

func (u *Uses) Validate(ctx context.Context) (errs *apis.FieldError) {
	if u == nil {
		return errs
	}
	if u.TaskRef.Name == "" {
		errs = errs.Also(apis.ErrMissingField("taskref.name"))
	}
	// TODO: Validate bindings
	return errs
}

func toUpstreamStep(steps []Step) []v1beta1.Step {
	s := make([]v1beta1.Step, len(steps))
	for i, step := range steps {
		s[i] = step.toUpstream()
	}
	return s
}
