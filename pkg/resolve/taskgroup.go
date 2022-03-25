package resolve

import (
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
)

type bindings struct {
	params     map[string]string
	workspaces map[string]string
}

func TaskSpec(spec *v1alpha1.TaskGroupSpec, usedTaskSpecs map[int]v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
	// TODO: Merge workspaces, results, volumes, sidecars, steptemplate
	taskSpec := &v1beta1.TaskSpec{
		Description:  spec.Description,
		Params:       spec.Params,
		Steps:        []v1beta1.Step{},
		Workspaces:   spec.Workspaces,
		Results:      spec.Results,
		Sidecars:     spec.Sidecars,
		StepTemplate: spec.StepTemplate,
	}
	usedTaskSpecsParams := []v1beta1.ParamSpec{}
	usedTaskSpecsWorkspaces := []v1beta1.WorkspaceDeclaration{}
	for i, step := range spec.Steps {
		if step.Uses != nil {
			usedTaskSpec, ok := usedTaskSpecs[i]
			if !ok {
				return taskSpec, fmt.Errorf("step %s uses not found", step.Name)
			}
			paramBindings := map[string]string{}
			if len(step.Uses.ParamBindings) > 0 {
				for _, b := range step.Uses.ParamBindings {
					paramBindings[b.Name] = b.Param
				}
				for _, p := range usedTaskSpec.Params {
					if _, ok := paramBindings[p.Name]; ok {
						continue
					}
					usedTaskSpecsParams = append(usedTaskSpecsParams, p)
				}
			} else {
				usedTaskSpecsParams = append(usedTaskSpecsParams, usedTaskSpec.Params...)
			}
			workspaceBindings := map[string]string{}
			if len(step.Uses.WorkspaceBindings) > 0 {
				for _, b := range step.Uses.WorkspaceBindings {
					workspaceBindings[b.Name] = b.Workspace
				}
				for _, p := range usedTaskSpec.Workspaces {
					if _, ok := workspaceBindings[p.Name]; ok {
						continue
					}
					usedTaskSpecsWorkspaces = append(usedTaskSpecsWorkspaces, p)
				}
			} else {
				usedTaskSpecsWorkspaces = append(usedTaskSpecsWorkspaces, usedTaskSpec.Workspaces...)
			}
			bindings := bindings{params: paramBindings, workspaces: workspaceBindings}
			rs, err := resolveSteps(step.Name, usedTaskSpec, bindings)
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
	workspaces, err := mergeWorkspaces(taskSpec.Workspaces, usedTaskSpecsWorkspaces)
	if err != nil {
		return taskSpec, err
	}
	taskSpec.Workspaces = workspaces

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

func mergeWorkspaces(tgWorkspaces, utWorkspaces []v1beta1.WorkspaceDeclaration) ([]v1beta1.WorkspaceDeclaration, error) {
	workspaces := []v1beta1.WorkspaceDeclaration{}
	seenWorkspaceNames := map[string]v1beta1.WorkspaceDeclaration{}
	for _, w := range tgWorkspaces {
		seenWorkspaceNames[w.Name] = w
		workspaces = append(workspaces, w)
	}
	for _, w := range utWorkspaces {
		if sw, ok := seenWorkspaceNames[w.Name]; ok {
			if sw.MountPath != w.MountPath {
				return workspaces, fmt.Errorf("Duplicate workspace %s with different mountPath", w.Name)
			}
			if sw.ReadOnly != w.ReadOnly {
				return workspaces, fmt.Errorf("Duplicate workspace %s with different readOnly option", w.Name)
			}
			if sw.Optional != w.Optional {
				return workspaces, fmt.Errorf("Duplicate workspace %s with different optional option", w.Name)
			}
			continue
		}
		workspaces = append(workspaces, w)
	}

	return workspaces, nil
}

func resolveSteps(name string, taskSpec v1beta1.TaskSpec, bindings bindings) ([]v1beta1.Step, error) {
	steps := make([]v1beta1.Step, len(taskSpec.Steps))
	var err error
	for i, s := range taskSpec.Steps {
		s, err = replaceParams(&s, bindings.params)
		if err != nil {
			return steps, nil
		}
		s, err = replaceWorkspaces(&s, bindings.workspaces)
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
		r[fmt.Sprintf("params['%s']", old)] = fmt.Sprintf("$(params['%s'])", new)
		r[fmt.Sprintf("params[\"%s\"]", old)] = fmt.Sprintf("$(params[\"%s\"])", new)
	}
	v1beta1.ApplyStepReplacements(s, r, map[string][]string{})
	return *s, nil
}

func replaceWorkspaces(s *v1beta1.Step, bindings map[string]string) (v1beta1.Step, error) {
	r := map[string]string{}
	for old, new := range bindings {
		// replace old with new
		r[fmt.Sprintf("workspaces.%s.path", old)] = fmt.Sprintf("$(workspaces.%s.path)", new)
		r[fmt.Sprintf("workspaces.%s.bound", old)] = fmt.Sprintf("$(workspaces.%s.bound)", new)
		r[fmt.Sprintf("workspaces.%s.claim", old)] = fmt.Sprintf("$(workspaces.%s.claim)", new)
		r[fmt.Sprintf("workspaces.%s.volume", old)] = fmt.Sprintf("$(workspaces.%s.volume)", new)
	}
	v1beta1.ApplyStepReplacements(s, r, map[string][]string{})
	return *s, nil
}