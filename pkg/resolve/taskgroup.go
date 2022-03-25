package resolve

import (
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
)

func TaskSpec(spec *v1alpha1.TaskGroupSpec, usedTaskSpecs map[int]v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
	// TODO: Merge params, workspaces, results, volumes, sidecars, steptemplate
	taskSpec := &v1beta1.TaskSpec{
		Description: spec.Description,
		Params:      spec.Params,
		Steps:       []v1beta1.Step{},
	}
	usedTaskSpecsParams := []v1beta1.ParamSpec{}
	for i, step := range spec.Steps {
		if step.Uses != nil {
			usedTaskSpec, ok := usedTaskSpecs[i]
			if !ok {
				return taskSpec, fmt.Errorf("step %s uses not found", step.Name)
			}
			paramBindings := map[string]string{}
			if len(step.Uses.ParamBindings) > 0 {
				// Remove params from Params
				for _, b := range step.Uses.ParamBindings {
					paramBindings[b.Name] = b.Param
				}
				for _, p := range usedTaskSpec.Params {
					if _, ok := paramBindings[p.Name]; ok {
						continue
					}
					usedTaskSpecsParams = append(usedTaskSpecsParams, p)
				}
				// Prepare for "replacement"
			} else {
				usedTaskSpecsParams = append(usedTaskSpecsParams, usedTaskSpec.Params...)
			}
			rs, err := resolveSteps(step.Name, paramBindings, usedTaskSpec)
			if err != nil {
				return taskSpec, err
			}
			taskSpec.Steps = append(taskSpec.Steps, rs...)
		} else {
			taskSpec.Steps = append(taskSpec.Steps, step.Step)
		}
	}
	params, err := mergeParams(taskSpec.Params, usedTaskSpecsParams)
	if err != nil {
		return taskSpec, err
	}
	taskSpec.Params = params

	return taskSpec, nil
}

func mergeParams(tgParams, utParams []v1beta1.ParamSpec) ([]v1beta1.ParamSpec, error) {
	params := []v1beta1.ParamSpec{}
	seenParamNames := map[string]v1beta1.ParamSpec{}
	for _, p := range tgParams {
		seenParamNames[p.Name] = p
		params = append(params, p)
	}
	for _, p := range utParams {
		if sp, ok := seenParamNames[p.Name]; ok {
			if sp.Type != p.Type {
				return params, fmt.Errorf("Duplicate param %s with different types", p.Name)
			}
			continue
		}
		params = append(params, p)
	}

	return params, nil
}

func resolveSteps(name string, paramBindings map[string]string, taskSpec v1beta1.TaskSpec) ([]v1beta1.Step, error) {
	steps := make([]v1beta1.Step, len(taskSpec.Steps))
	for i, s := range taskSpec.Steps {
		s, err := replaceParams(&s, paramBindings)
		if err != nil {
			return steps, nil
		}
		steps[i] = s
		steps[i].Name = name + "-" + s.Name
	}
	return steps, nil
}

func replaceParams(s *v1beta1.Step, bindings map[string]string) (v1beta1.Step, error) {
	r := map[string]string{}
	for old, new := range bindings {
		// replace old with new
		r[fmt.Sprintf("params.%s", old)] = fmt.Sprintf("$(params.%s)", new)
	}
	v1beta1.ApplyStepReplacements(s, r, map[string][]string{})
	return *s, nil
}
