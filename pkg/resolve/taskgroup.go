package resolve

import (
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/vdemeester/tekton-task-group/pkg/apis/taskgroup/v1alpha1"
)

func TaskSpec(spec *v1alpha1.TaskGroupSpec, usedTaskSpecs map[int]v1beta1.TaskSpec) (*v1beta1.TaskSpec, error) {
	// TODO: Merge params, workspaces, results, volumes, sidecars, steptemplate
	taskSpec := &v1beta1.TaskSpec{
		Params: []v1beta1.ParamSpec{},
		Steps:  []v1beta1.Step{},
	}
	for i, step := range spec.Steps {
		if step.Uses != nil {
			usedTaskSpec, ok := usedTaskSpecs[i]
			if !ok {
				return taskSpec, fmt.Errorf("step %s uses not found", step.Name)
			}
			taskSpec.Steps = append(taskSpec.Steps, resolveSteps(step.Name, usedTaskSpec)...)
		} else {
			taskSpec.Steps = append(taskSpec.Steps, step.Step)
		}
	}

	return taskSpec, nil
}

func resolveSteps(name string, taskSpec v1beta1.TaskSpec) []v1beta1.Step {
	steps := make([]v1beta1.Step, len(taskSpec.Steps))
	for i, s := range taskSpec.Steps {
		steps[i] = s
		steps[i].Name = name + "-" + s.Name
	}
	return steps
}
