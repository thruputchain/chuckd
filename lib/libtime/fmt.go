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
		return fmt.Sprintf("%dmin", int64(d.Minutes()))
	case d >= time.Second:
		return fmt.Sprintf("%dsec", int64(d.Seconds()))
	case d >= time.Millisecond:
		return fmt.Sprintf("%dmsec", int64(d.Milliseconds()))
	case d >= time.Microsecond:
		return fmt.Sprintf("%dµsec", int64(d.Microseconds()))
	default:
		return fmt.Sprintf("%dnsec", int64(d.Nanoseconds()))
	}
}
