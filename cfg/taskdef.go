package cfg

import (
	"fmt"
	"strings"

	"github.com/simplesurance/baur/cfg/resolver"
)

type TaskDef interface {
	GetCommand() string
	GetIncludes() *[]string
	GetInput() *Input
	GetName() string
	GetOutput() *Output
}

// TaskMerge loads the includes of the task and merges them with the task itself.
func TaskMerge(task TaskDef, workingDir string, resolver resolver.Resolver, includeDB *IncludeDB) error {
	for _, includeSpec := range *task.GetIncludes() {
		inputInclude, err := includeDB.LoadInputInclude(resolver, workingDir, includeSpec)
		if err == nil {
			task.GetInput().Files.Merge(&inputInclude.Files)
			task.GetInput().GitFiles.Merge(&inputInclude.GitFiles)
			task.GetInput().GolangSources.Merge(&inputInclude.GolangSources)

			continue
		}

		if err != nil && err != ErrIncludeIDNotFound {
			return err
		}

		outputInclude, err := includeDB.LoadOutputInclude(resolver, workingDir, includeSpec)
		if err != nil {
			if err == ErrIncludeIDNotFound {
				return fmt.Errorf("input or output include %q does not exist", includeSpec)
			}

			return err
		}

		task.GetOutput().Merge(outputInclude)
	}

	return nil
}

// TaskValidate validates the task section
func TaskValidate(t TaskDef) error {
	if len(t.GetCommand()) == 0 {
		return NewFieldError("can not be empty", "command")
	}

	if strings.Contains(t.GetName(), ".") {
		return NewFieldError("dots are not allowed in task names", "name")
	}

	if err := validateIncludes(*t.GetIncludes()); err != nil {
		return FieldErrorWrap(err, "includes")
	}

	if t.GetInput() == nil {
		return NewFieldError("section is empty", "Input")
	}

	if err := InputValidate(t.GetInput()); err != nil {
		return FieldErrorWrap(err, "Input")
	}

	if t.GetOutput() == nil {
		return nil
	}

	if err := OutputValidate(t.GetOutput()); err != nil {
		return FieldErrorWrap(err, "Output")
	}

	return nil
}
