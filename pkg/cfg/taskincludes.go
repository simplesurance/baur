package cfg

import "github.com/simplesurance/baur/v1/pkg/cfg/resolver"

type TaskIncludes []*TaskInclude

func (tasks TaskIncludes) validate() error {
	for _, task := range tasks {
		if err := task.validate(); err != nil {
			if task.Name != "" {
				err = fieldErrorWrap(err, "Task")
			}

			return err
		}
	}

	return nil
}

func (tasks TaskIncludes) merge(workingDir string, resolver resolver.Resolver, db *IncludeDB) error {
	for _, task := range tasks {
		err := taskMerge(task, workingDir, resolver, db)
		if err != nil {
			if task.Name != "" {
				err = fieldErrorWrap(err, task.Name)
			}

			return err
		}
	}

	return nil
}
