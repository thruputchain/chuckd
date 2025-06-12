package liblog

import (
	"testing"

	"github.com/rs/zerolog"
)

func TestLogger(t *testing.T) {
	LogLevel(zerolog.DebugLevel)
	Debugf("debug message")
}
