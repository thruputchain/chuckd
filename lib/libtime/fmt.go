package libtime

import (
	"fmt"
	"time"
)

func FmtDur(d time.Duration) string {
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%.1fh", d.Hours())
	case d >= time.Minute:
		return fmt.Sprintf("%dm", int64(d.Minutes()))
	case d >= time.Second:
		return fmt.Sprintf("%ds", int64(d.Seconds()))
	case d >= time.Millisecond:
		return fmt.Sprintf("%dms", int64(d.Milliseconds()))
	case d >= time.Microsecond:
		return fmt.Sprintf("%dµs", int64(d.Microseconds()))
	default:
		return fmt.Sprintf("%dns", int64(d.Nanoseconds()))
	}
}
