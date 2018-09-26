package flag

import (
	"time"
)

const (
	dateTimeFormat  = "2006.01.02-15:04"
	dateTimeFormat1 = "2006.01.02-15:04:05-MST"

	// DateTimeExampleFormat is an exemplary valid datetime flag
	DateTimeExampleFormat = "2006.01.28-15:30"
	// DateTimeFormatDescr contains a description of the supported formats
	DateTimeFormatDescr = "YYYY.MM.DD-HH:MM[:SS-TZ]"
)

// DateTimeFlagValue is the DateTime pflag flag
type DateTimeFlagValue struct {
	time.Time
}

// String returns an empty string
func (v *DateTimeFlagValue) String() string {
	return ""
}

// Set implements the pflag.Value interface
func (v *DateTimeFlagValue) Set(timeStr string) error {
	var t time.Time
	var err error

	t, err = time.Parse(dateTimeFormat, timeStr)
	if err != nil {
		t, err = time.Parse(dateTimeFormat1, timeStr)
		if err != nil {
			return err
		}
	}

	v.Time = t

	return nil
}

// Type returns the value description string
func (*DateTimeFlagValue) Type() string {
	return "<datetime>"
}
