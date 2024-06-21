package flag

import (
	"time"
)

const (
	// DateTimeFormat describes a short format supported by the flag
	DateTimeFormat = "2006.01.02-15:04"
	// DateTimeFormatTz describes long format supported by the flag
	DateTimeFormatTz = "2006.01.02-15:04:05-MST"

	// DateTimeExampleFormat is an exemplary valid datetime flag
	DateTimeExampleFormat = "2006.01.28-15:30"
	// DateTimeFormatDescr contains a description of the supported formats
	DateTimeFormatDescr = "YYYY.MM.DD-HH:MM[:SS-TZ]"
)

// DateTimeFlagValue is the DateTime pflag flag
type DateTimeFlagValue struct {
	time.Time
	IsSet bool
}

// String returns the default value in the usage output
func (v *DateTimeFlagValue) String() string {
	return ""
}

// Set implements the pflag.Value interface
func (v *DateTimeFlagValue) Set(timeStr string) error {
	var t time.Time
	var err error

	t, err = time.ParseInLocation(DateTimeFormat, timeStr, time.Local)
	if err != nil {
		t, err = time.ParseInLocation(DateTimeFormatTz, timeStr, time.Local)
		if err != nil {
			return err
		}
	}

	v.Time = t
	v.IsSet = true

	return nil
}

// Type returns the value description string
func (*DateTimeFlagValue) Type() string {
	return "DATETIME"
}
