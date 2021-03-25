package cfg

import (
	"fmt"
	"strings"

	"github.com/simplesurance/baur/v2/pkg/cfg/resolver"
)

type taskDef interface {
	GetCommand() []string
	GetIncludes() *[]string
	GetInput() *Input
	GetName() string
	GetOutput() *Output
	addCfgFilepath(path string)
}

// taskMerge loads the includes of the task and merges them with the task itself.
func taskMerge(task taskDef, workingDir string, resolver Resolver, includeDB *IncludeDB) error {
	for _, includeSpec := range *task.GetIncludes() {
		inputInclude, err := includeDB.loadInputInclude(resolver, workingDir, includeSpec)
		if err == nil {
			inputInclude = inputInclude.clone()
			task.GetInput().merge(inputInclude)
			task.addCfgFilepath(inputInclude.filepath)

			continue
		}

		// The includeSpec can refer to an input or output.
		// If no input include for it exist, ErrIncludeIDNotFound is
		// ignored and we try to load an output include instead.
		if err != nil && err != ErrIncludeIDNotFound {
			return fieldErrorWrap(fmt.Errorf("%q: %w", includeSpec, err), "Includes")
		}

		outputInclude, err := includeDB.loadOutputInclude(resolver, workingDir, includeSpec)
		if err != nil {
			if err == ErrIncludeIDNotFound {
				return fieldErrorWrap(fmt.Errorf("%q: %w", includeSpec, err), "Includes")
			}

			return err
		}

		outputInclude = outputInclude.clone()
		task.GetOutput().Merge(outputInclude)

		task.addCfgFilepath(outputInclude.filepath)
	}

	return nil
}

// taskValidate validates the task section
func taskValidate(t taskDef) error {
	if len(t.GetCommand()) == 0 {
		return newFieldError("can not be empty", "command")
	}

	if t.GetName() == "" {
		return newFieldError("name can not be empty", "name")
	}

	if strings.Contains(t.GetName(), ".") {
		return newFieldError("dots are not allowed in task names", "name")
	}

	if err := validateIncludes(*t.GetIncludes()); err != nil {
		return fieldErrorWrap(err, "includes")
	}

	if t.GetInput() == nil {
		return newFieldError("section is empty", "Input")
	}

	if err := inputValidate(t.GetInput()); err != nil {
		return fieldErrorWrap(err, "Input")
	}

	if t.GetOutput() == nil {
		return nil
	}

	if err := outputValidate(t.GetOutput()); err != nil {
		return fieldErrorWrap(err, "Output")
	}

	return nil
}
