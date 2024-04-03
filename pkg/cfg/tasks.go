package cfg

import (
	"fmt"
	"slices"
)

type Tasks []*Task

func (tasks Tasks) resolve(resolver Resolver) error {
	for _, t := range tasks {
		if err := t.resolve(resolver); err != nil {
			return fieldErrorWrap(err, "Tasks", t.Name)
		}
	}

	return nil
}

func (tasks Tasks) validate() error {
	duplMap := make(map[string]struct{}, len(tasks))

	for _, task := range tasks {
		err := taskValidate(task)
		if err != nil {
			if task.Name != "" {
				return fieldErrorWrap(err, "Task", task.Name)
			}

			return fieldErrorWrap(err, "Task")
		}

		_, exist := duplMap[task.Name]
		if exist {
			return newFieldError(
				fmt.Sprintf("multiple tasks with name '%s' exist, task names must be unique", task.Name),
				"Task",
			)
		}
		duplMap[task.Name] = struct{}{}
	}

	return tasks.validateTaskInfosAreCycleFree()
}

func (tasks Tasks) validateTaskInfosAreCycleFree() error {
	allTasks := make(map[string]*Task)
	for _, task := range tasks {
		allTasks[task.Name] = task
	}

	for _, task := range tasks {
		if err := task.validateTaskInfoAresCycleFree(allTasks, nil); err != nil {
			return fieldErrorWrap(err, elementPathWithID("Task", task.Name))
		}
	}

	return nil
}

func (task *Task) validateTaskInfoAresCycleFree(allTasks map[string]*Task, recursionTracker []string) error {
	recursionTracker = append(recursionTracker, task.Name)
	for _, ti := range task.Input.taskInfos() {
		if slices.Contains(recursionTracker, ti.TaskName) {
			return newFieldError(
				"TaskInfo dependency is cyclic",
				elementPathWithID("TaskInfo", ti.TaskName), "task_id",
			)
		}

		task, exists := allTasks[ti.TaskName]
		if !exists {
			return newFieldError(
				fmt.Sprintf("a task named %q does not exists", ti.TaskName),
				"Inputs", elementPathWithID("TaskInfo", ti.TaskName), "task_id",
			)
		}

		err := task.validateTaskInfoAresCycleFree(allTasks, append(slices.Clone(recursionTracker), ti.TaskName))
		if err != nil {
			return fieldErrorWrap(err, "Inputs", elementPathWithID("TaskInfo", ti.TaskName))
		}
	}

	return nil
}
