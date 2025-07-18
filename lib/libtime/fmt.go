package libtime

import (
	"fmt"
	"time"
)

func FmtDur(d time.Duration) string {
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%.1f h", d.Hours())
	case d >= time.Minute:
		return fmt.Sprintf("%d min", int64(d.Minutes()))
	case d >= time.Second:
		return fmt.Sprintf("%d sec", int64(d.Seconds()))
	case d >= time.Millisecond:
		return fmt.Sprintf("%d msec", int64(d.Milliseconds()))
	case d >= time.Microsecond:
		return fmt.Sprintf("%d µsec", int64(d.Microseconds()))
	default:
		return fmt.Sprintf("%d nsec", int64(d.Nanoseconds()))
	}
}
