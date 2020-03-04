package flag

import (
	"errors"
	"fmt"
	"strings"

	"github.com/simplesurance/baur"
)

// Valid commandline values
const (
	taskStatusExist          = "exist"
	taskStatusPending        = "pending"
	taskStatusInputUndefined = "inputs-undefined"
)

// TaskStatusFormatDescription is the format description for the flag
const TaskStatusFormatDescription string = "one of " +
	taskStatusExist + ", " +
	taskStatusPending + ", " +
	taskStatusInputUndefined

// TaskStatus is a commandline parameter to specify build status filters
type TaskStatus struct {
	Status baur.BuildStatus
	isSet  bool
}

// String returns the default value in the usage output
func (b *TaskStatus) String() string {
	return ""
}

// Set parses the passed string and sets the SortFlagValue
func (b *TaskStatus) Set(val string) error {
	b.isSet = true

	switch strings.ToLower(val) {
	case taskStatusExist:
		b.Status = baur.BuildStatusExist
	case taskStatusPending:
		b.Status = baur.BuildStatusPending
	case taskStatusInputUndefined:
		b.Status = baur.BuildStatusInputsUndefined

	default:
		return errors.New("status must be " + TaskStatusFormatDescription)
	}

	return nil
}

// Type returns the format description of the flag
func (b *TaskStatus) Type() string {
	return "<STATUS>"
}

// Usage returns a usage description, important parts are passed through
// highlightFn
func (b *TaskStatus) Usage(highlightFn func(a ...interface{}) string) string {
	return strings.TrimSpace(fmt.Sprintf(`
Only show tasks with this status
Format: %s
where %s is one of: %s, %s, %s`,
		highlightFn(b.Type()),
		highlightFn("STATUS"),
		highlightFn(taskStatusExist),
		highlightFn(taskStatusPending),
		highlightFn(taskStatusInputUndefined),
	))
}

// IsSet returns true if the flag parsed a commandline value (Set() was called)
func (b *TaskStatus) IsSet() bool {
	return b.isSet
}
