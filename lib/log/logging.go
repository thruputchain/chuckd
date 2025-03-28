package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var verbose bool

func init() {
	LogQuietly()
}

func IsVerbose() bool {
	return verbose
}

func LogQuietly() {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out:        os.Stderr,
			NoColor:    true,
			TimeFormat: time.RFC3339,
		},
	)
	verbose = false
}

func LogVerbosely() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out:        os.Stderr,
			NoColor:    false,
			TimeFormat: time.RFC3339,
		},
	)
	verbose = true
}

func Writer() io.Writer {
	return log.Logger
}

func Debugf(template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	log.Debug().Msg(src + msg)
}

func Infof(template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	log.Info().Msg(src + msg)
}

func Fatalf(template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	log.Fatal().Msg(src + msg)
}

func Errorf(template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	log.Error().Msg(src + msg)
}

func Warnf(template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	log.Warn().Msg(src + msg)
}

func Info() *zerolog.Event {
	return log.Info()
}

func Debug() *zerolog.Event {
	return log.Debug()
}

func Fatal() *zerolog.Event {
	return log.Fatal()
}

func Error() *zerolog.Event {
	return log.Error()
}

func Warn() *zerolog.Event {
	return log.Warn()
}
