package cfg

import "github.com/simplesurance/baur/v1/cfg/resolver"

type TaskIncludes []*TaskInclude

func (tasks TaskIncludes) Validate() error {
	for _, task := range tasks {
		if err := task.Validate(); err != nil {
			if task.Name != "" {
				err = FieldErrorWrap(err, "Task")
			}

			return err
		}
	}

	return nil
}

func (tasks TaskIncludes) Merge(workingDir string, resolver resolver.Resolver, db *IncludeDB) error {
	for _, task := range tasks {
		if err := TaskMerge(task, workingDir, resolver, db); err != nil {
			if task.Name != "" {
				err = FieldErrorWrap(err, task.Name)
			}

			return err
		}
	}

	return nil
}
