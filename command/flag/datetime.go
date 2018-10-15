package flag

import (
	"time"

	"github.com/pkg/errors"
)

// InputTimeFormat is the format for datetime inputs in flags
const InputTimeFormat = "2006-01-02T15:04:05"

// DateTimeFlagValue is the DateTime pflag flag
type DateTimeFlagValue struct {
	time.Time
}

// String returns the string value of a datetime flag
func (v *DateTimeFlagValue) String() string {
	return ""
}

// Set implements the pflag.Value interface
func (v *DateTimeFlagValue) Set(timeStr string) error {
	t, err := time.Parse(InputTimeFormat, timeStr)
	if err != nil {
		return errors.Wrap(err, "error while parsing time")
	}

	v.Time = t

	return nil
}

// Type returns the name of this type
func (*DateTimeFlagValue) Type() string {
	return "datetime"
}
