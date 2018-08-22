package flag

import (
	"github.com/pkg/errors"
	"time"
)

const InputTimeFormat = "2006-01-02T15:04:05"

type DateTimeFlagValue struct {
	time.Time
}

func (v *DateTimeFlagValue) String() string {
	return ""
}

func (v *DateTimeFlagValue) Set(timeStr string) error {
	t, err := time.Parse(InputTimeFormat, timeStr)
	if err != nil {
		return errors.Wrap(err, "error while parsing time")
	}

	v.Time = t

	return nil
}

func (*DateTimeFlagValue) Type() string {
	return "datetime"
}
