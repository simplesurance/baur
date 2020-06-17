package term

import (
	"fmt"
	"time"
)

func BytesToMib(bytes uint64) string {
	return fmt.Sprintf("%.3f", float64(bytes)/1024/1024)
}

func DurationToStrSeconds(duration time.Duration) string {
	return fmt.Sprintf("%.3f", duration.Seconds())
}

func StrDurationSec(start, end time.Time) string {
	return DurationToStrSeconds(end.Sub(start))
}
