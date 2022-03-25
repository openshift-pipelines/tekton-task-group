package resolve

import (
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type stepTransformer func(*v1beta1.Step) (*v1beta1.Step, error)

type taskSpecTransformer func(*v1beta1.TaskSpec) (*v1beta1.TaskSpec, error)

type bindings struct {
	params     map[string]string
	workspaces map[string]string
}

func TaskSpec(spec *v1alpha1.TaskGroupSpec, usedTaskSpecs map[int]v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
	// TODO: Merge results, volumes, sidecars, steptemplate
	taskSpec := &v1beta1.TaskSpec{
		Description:  spec.Description,
		Params:       spec.Params,
		Steps:        []v1beta1.Step{},
		Workspaces:   spec.Workspaces,
		Results:      spec.Results,
		Sidecars:     spec.Sidecars,
		StepTemplate: spec.StepTemplate,
		Volumes:      spec.Volumes,
	}
	usedTaskSpecsParams := []v1beta1.ParamSpec{}
	usedTaskSpecsWorkspaces := []v1beta1.WorkspaceDeclaration{}
	usedTaskSpecResults := []v1beta1.TaskResult{}
	usedTaskSpecSidecars := []v1beta1.Sidecar{}
	usedTaskSpecVolumes := []corev1.Volume{}
	for i, step := range spec.Steps {
		if step.Uses != nil {
			usedTaskSpec, ok := usedTaskSpecs[i]
			if !ok {
				return taskSpec, fmt.Errorf("step %s uses not found", step.Name)
			}
			// Params
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
			// Workspaces
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
			// Results
			usedTaskSpecResults = append(usedTaskSpecResults, usedTaskSpec.Results...)
			// Sidecars
			usedTaskSpecSidecars = append(usedTaskSpecSidecars, usedTaskSpec.Sidecars...)
			// Volumes
			usedTaskSpecVolumes = append(usedTaskSpecVolumes, usedTaskSpec.Volumes...)
			// Step
			stepName := step.Name
			if stepName == "" {
				stepName = fmt.Sprintf("unamed-%d", i)
			}
			rs, err := resolveSteps(stepName, usedTaskSpec,
				replaceParams(paramBindings),
				replaceWorkspaces(workspaceBindings),
			)
			if err != nil {
				return taskSpec, err
			}
			taskSpec.Steps = append(taskSpec.Steps, rs...)
		} else {
			taskSpec.Steps = append(taskSpec.Steps, step.Step)
		}
	}
	for _, t := range []taskSpecTransformer{
		mergeParams(usedTaskSpecsParams),
		mergeWorkspaces(usedTaskSpecsWorkspaces),
		mergeResults(usedTaskSpecResults),
		mergeSidecars(usedTaskSpecSidecars),
		mergeVolumes(usedTaskSpecVolumes),
	} {
		taskSpec, err := t(taskSpec)
		if err != nil {
			return taskSpec, err
		}
	}

	return taskSpec, nil
}

func mergeParams(utParams []v1beta1.ParamSpec) taskSpecTransformer {
	return func(spec *v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
		params := []v1beta1.ParamSpec{}
		seenParamNames := map[string]v1beta1.ParamSpec{}
		for _, p := range spec.Params {
			seenParamNames[p.Name] = p
			params = append(params, p)
		}
		for _, p := range utParams {
			if sp, ok := seenParamNames[p.Name]; ok {
				if sp.Type != p.Type {
					return spec, fmt.Errorf("Duplicate param %s with different types", p.Name)
				}
				continue
			}
			params = append(params, p)
		}
		spec.Params = params
		return spec, nil
	}
}

func mergeWorkspaces(utWorkspaces []v1beta1.WorkspaceDeclaration) taskSpecTransformer {
	return func(spec *v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
		workspaces := []v1beta1.WorkspaceDeclaration{}
		seenWorkspaceNames := map[string]v1beta1.WorkspaceDeclaration{}
		for _, w := range spec.Workspaces {
			seenWorkspaceNames[w.Name] = w
			workspaces = append(workspaces, w)
		}
		for _, w := range utWorkspaces {
			if sw, ok := seenWorkspaceNames[w.Name]; ok {
				if sw.MountPath != w.MountPath {
					return spec, fmt.Errorf("Duplicate workspace %s with different mountPath", w.Name)
				}
				if sw.ReadOnly != w.ReadOnly {
					return spec, fmt.Errorf("Duplicate workspace %s with different readOnly option", w.Name)
				}
				if sw.Optional != w.Optional {
					return spec, fmt.Errorf("Duplicate workspace %s with different optional option", w.Name)
				}
				continue
			}
			workspaces = append(workspaces, w)
		}
		spec.Workspaces = workspaces
		return spec, nil
	}
}

func mergeResults(utResults []v1beta1.TaskResult) taskSpecTransformer {
	return func(spec *v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
		results := []v1beta1.TaskResult{}
		seenResultNames := map[string]v1beta1.TaskResult{}
		for _, r := range spec.Results {
			seenResultNames[r.Name] = r
			results = append(results, r)
		}
		for _, r := range utResults {
			if _, ok := seenResultNames[r.Name]; ok {
				continue
			}
			results = append(results, r)
		}
		spec.Results = results

		return spec, nil
	}
}

func mergeSidecars(utSidecars []v1beta1.Sidecar) taskSpecTransformer {
	return func(spec *v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
		sidecars := []v1beta1.Sidecar{}
		seenSidecarNames := map[string]v1beta1.Sidecar{}
		for _, s := range spec.Sidecars {
			seenSidecarNames[s.Name] = s
			sidecars = append(sidecars, s)
		}
		for _, s := range utSidecars {
			if _, ok := seenSidecarNames[s.Name]; ok {
				return spec, fmt.Errorf("Duplicated sidecars: %s", s.Name)
			}
			sidecars = append(sidecars, s)
		}
		spec.Sidecars = sidecars

		return spec, nil
	}
}

func mergeVolumes(utVolumes []corev1.Volume) taskSpecTransformer {
	return func(spec *v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
		volumes := []corev1.Volume{}
		seenVolumeNames := map[string]corev1.Volume{}
		for _, v := range spec.Volumes {
			seenVolumeNames[v.Name] = v
			volumes = append(volumes, v)
		}
		for _, v := range utVolumes {
			if _, ok := seenVolumeNames[v.Name]; ok {
				return spec, fmt.Errorf("Duplicated volumes: %s", v.Name)
			}
			volumes = append(volumes, v)
		}
		spec.Volumes = volumes

		return spec, nil
	}
}

func resolveSteps(name string, taskSpec v1beta1.TaskSpec, transformers ...stepTransformer) ([]v1beta1.Step, error) {
	steps := make([]v1beta1.Step, len(taskSpec.Steps))
	var err error
	for i, s := range taskSpec.Steps {
		ps := &s
		for _, t := range transformers {
			ps, err = t(ps)
			if err != nil {
				return steps, err
			}
		}
		steps[i] = *ps
		steps[i].Name = name + "-" + s.Name
	}
	return steps, nil
}

func replaceParams(bindings map[string]string) stepTransformer {
	return func(s *v1beta1.Step) (*v1beta1.Step, error) {
		r := map[string]string{}
		for old, new := range bindings {
			// replace old with new
			r[fmt.Sprintf("params.%s", old)] = fmt.Sprintf("$(params.%s)", new)
			r[fmt.Sprintf("params['%s']", old)] = fmt.Sprintf("$(params['%s'])", new)
			r[fmt.Sprintf("params[\"%s\"]", old)] = fmt.Sprintf("$(params[\"%s\"])", new)
		}
		v1beta1.ApplyStepReplacements(s, r, map[string][]string{})
		return s, nil
	}
}

func replaceWorkspaces(bindings map[string]string) stepTransformer {
	return func(s *v1beta1.Step) (*v1beta1.Step, error) {
		r := map[string]string{}
		for old, new := range bindings {
			// replace old with new
			r[fmt.Sprintf("workspaces.%s.path", old)] = fmt.Sprintf("$(workspaces.%s.path)", new)
			r[fmt.Sprintf("workspaces.%s.bound", old)] = fmt.Sprintf("$(workspaces.%s.bound)", new)
			r[fmt.Sprintf("workspaces.%s.claim", old)] = fmt.Sprintf("$(workspaces.%s.claim)", new)
			r[fmt.Sprintf("workspaces.%s.volume", old)] = fmt.Sprintf("$(workspaces.%s.volume)", new)
		}
		v1beta1.ApplyStepReplacements(s, r, map[string][]string{})
		return s, nil
	}
}
