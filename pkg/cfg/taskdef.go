package cfg

import (
	"errors"
	"fmt"
	"strings"
)

type taskDef interface {
	command() []string
	includes() *[]string
	input() *Input
	name() string
	output() *Output
	addCfgFilepath(path string)
}

// taskMerge loads the includes of the task and merges them with the task itself.
func taskMerge(task taskDef, workingDir string, resolver Resolver, includeDB *IncludeDB) error {
	for _, includeSpec := range *task.includes() {
		inputInclude, err := includeDB.loadInputInclude(resolver, workingDir, includeSpec)
		if err == nil {
			inputInclude = inputInclude.clone()
			task.input().merge(inputInclude)
			task.addCfgFilepath(inputInclude.filepath)

			continue
		}

		// The includeSpec can refer to an input or output.
		// If no input include for it exist, ErrIncludeIDNotFound is
		// ignored and we try to load an output include instead.
		if err != nil && !errors.Is(err, ErrIncludeIDNotFound) {
			return fieldErrorWrap(fmt.Errorf("%q: %w", includeSpec, err), "Includes")
		}

		outputInclude, err := includeDB.loadOutputInclude(resolver, workingDir, includeSpec)
		if err != nil {
			if errors.Is(err, ErrIncludeIDNotFound) {
				return fieldErrorWrap(fmt.Errorf("%q: %w", includeSpec, err), "Includes")
			}

			return err
		}

		outputInclude = outputInclude.clone()
		task.output().Merge(outputInclude)

		task.addCfgFilepath(outputInclude.filepath)
	}

	return nil
}

// taskValidate validates the task section
func taskValidate(t taskDef) error {
	if len(t.command()) == 0 {
		return newFieldError("can not be empty", "command")
	}

	if err := validateTaskOrAppName(t.name()); err != nil {
		return fieldErrorWrap(err, "name")
	}

	if strings.Contains(t.name(), ".") {
		return newFieldError("dots are not allowed in task names", "name")
	}

	if err := validateIncludes(*t.includes()); err != nil {
		return fieldErrorWrap(err, "includes")
	}

	if t.input() == nil {
		return newFieldError("section is empty", "Input")
	}

	if err := inputValidate(t.input()); err != nil {
		return fieldErrorWrap(err, "Input")
	}

	if t.output() == nil {
		return nil
	}

	if err := outputValidate(t.output()); err != nil {
		return fieldErrorWrap(err, "Output")
	}

	return nil
}
