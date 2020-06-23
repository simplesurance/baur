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

func FormatDuration(d time.Duration) string {
	if d.Minutes() > 1 {
		return d.Round(time.Second).String()
	}

	if d.Milliseconds() > 1 {
		return d.Round(time.Millisecond).String()
	}

	return d.String()
}
