package term

import (
	"fmt"
	"time"
)

type FormatOption func(*fmtSettings)

type fmtSettings struct {
	baseUnitWithoutUnitName bool
}

// FormatBaseWithoutUnitName when enabled the Format functions return the value
// in its base unit without unit-name suffix
func FormatBaseWithoutUnitName(enable bool) FormatOption {
	return func(o *fmtSettings) {
		o.baseUnitWithoutUnitName = enable
	}
}

func evalOptions(opts []FormatOption) *fmtSettings {
	var s fmtSettings

	for _, opt := range opts {
		opt(&s)
	}

	return &s
}
func FormatSize(bytes uint64, opts ...FormatOption) string {
	if evalOptions(opts).baseUnitWithoutUnitName {
		return fmt.Sprint(bytes)
	}

	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	if bytes < 1024*1024 {
		return fmt.Sprintf("%.3f KiB", float64(bytes)/1024)
	}

	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.3f MiB", float64(bytes)/1024/1024)
	}

	return fmt.Sprintf("%.3f GiB", float64(bytes)/1024/1024/1024)
}

func FormatDuration(d time.Duration, opts ...FormatOption) string {
	if evalOptions(opts).baseUnitWithoutUnitName {
		return fmt.Sprintf("%.3f", d.Seconds())
	}

	if d.Minutes() >= 1 {
		return d.Truncate(time.Second).String()
	}

	if d.Seconds() >= 1 {
		return d.Truncate(time.Millisecond).String()
	}

	return d.Truncate(time.Microsecond).String()
}
