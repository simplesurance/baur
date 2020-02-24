package flag

import (
	"errors"
	"fmt"
	"strings"

	"github.com/simplesurance/baur"
)

// Valid commandline values
const (
	buildStatusExist          = "exist"
	buildStatusPending        = "pending"
	buildStatusInputUndefined = "inputs-undefined"
)

// BuildStatusFormatDescription is the format description for the flag
const BuildStatusFormatDescription string = "one of " +
	buildStatusExist + ", " +
	buildStatusPending + ", " +
	buildStatusInputUndefined

// BuildStatus is a commandline parameter to specify build status filters
type BuildStatus struct {
	Status baur.BuildStatus
	isSet  bool
}

// String returns the default value in the usage output
func (b *BuildStatus) String() string {
	return ""
}

// Set parses the passed string and sets the SortFlagValue
func (b *BuildStatus) Set(val string) error {
	b.isSet = true

	switch strings.ToLower(val) {
	case buildStatusExist:
		b.Status = baur.BuildStatusExist
	case buildStatusPending:
		b.Status = baur.BuildStatusPending
	case buildStatusInputUndefined:
		b.Status = baur.BuildStatusPending
	default:
		return errors.New("status must be " + BuildStatusFormatDescription)
	}

	return nil
}

// Type returns the format description of the flag
func (b *BuildStatus) Type() string {
	return "<STATUS>"
}

// Usage returns a usage description, important parts are passed through
// highlightFn
func (b *BuildStatus) Usage(highlightFn func(a ...interface{}) string) string {
	return strings.TrimSpace(fmt.Sprintf(`
Only show applications with this build status
Format: %s
where %s is one of: %s, %s, %s`,
		highlightFn(b.Type()),
		highlightFn("STATUS"),
		highlightFn(buildStatusExist),
		highlightFn(buildStatusPending),
		highlightFn(buildStatusInputUndefined),
	))
}

// IsSet returns true if the flag parsed a commandline value (Set() was called)
func (b *BuildStatus) IsSet() bool {
	return b.isSet
}
