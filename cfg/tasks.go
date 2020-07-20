package cfg

import (
	"fmt"

	"github.com/simplesurance/baur/v1/cfg/resolver"
)

type Tasks []*Task

func (tasks Tasks) Merge(workingDir string, resolver resolver.Resolver, includedb *IncludeDB) error {
	for _, task := range tasks {
		if err := TaskMerge(task, workingDir, resolver, includedb); err != nil {
			return FieldErrorWrap(err, "Task")
		}
	}

	return nil
}

func (tasks Tasks) Resolve(resolvers resolver.Resolver) error {
	for _, t := range tasks {
		if err := t.Resolve(resolvers); err != nil {
			return FieldErrorWrap(err, "Tasks", t.Name)
		}
	}

	return nil
}

func (tasks Tasks) Validate() error {
	duplMap := make(map[string]struct{}, len(tasks))

	for _, task := range tasks {
		_, exist := duplMap[task.Name]
		if exist {
			return NewFieldError(
				fmt.Sprintf("multiple tasks with name '%s' exist, task names must be unique", task.Name),
				"Task",
			)
		}
		duplMap[task.Name] = struct{}{}

		err := TaskValidate(task)
		if err != nil {
			return FieldErrorWrap(err, "Task", task.Name)
		}
	}

	return nil
}
