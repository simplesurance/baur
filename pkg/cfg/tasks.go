package cfg

import (
	"fmt"

	"github.com/simplesurance/baur/v2/pkg/cfg/resolver"
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

	return nil
}
