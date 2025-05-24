package liblog

import "testing"

func TestLogger(t *testing.T) {
	LogVerbosely()
	Debugf("debug message")
}
