package term

import (
	"fmt"
	"time"
)

func FormatSize(bytes uint64) string {
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

func DurationToStrSeconds(duration time.Duration) string {
	return fmt.Sprintf("%.3f", duration.Seconds())
}

func StrDurationSec(start, end time.Time) string {
	return DurationToStrSeconds(end.Sub(start))
}
