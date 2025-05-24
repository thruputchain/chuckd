package liblog

import (
	"context"
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
			NoColor:    true,
			TimeFormat: time.RFC3339,
		},
	)
	verbose = true
}

// global logger

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

// ctx logger

func CtxWriter(ctx context.Context) io.Writer {
	return zerolog.Ctx(ctx)
}

func CtxDebugf(ctx context.Context, template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	zerolog.Ctx(ctx).Debug().Msg(src + msg)
}

func CtxInfof(ctx context.Context, template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	zerolog.Ctx(ctx).Info().Msg(src + msg)
}

func CtxFatalf(ctx context.Context, template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	zerolog.Ctx(ctx).Fatal().Msg(src + msg)
}

func CtxErrorf(ctx context.Context, template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	zerolog.Ctx(ctx).Error().Msg(src + msg)
}

func CtxWarnf(ctx context.Context, template string, args ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	src := fmt.Sprintf("%s:%d ", file, line)
	msg := fmt.Sprintf(template, args...)
	zerolog.Ctx(ctx).Warn().Msg(src + msg)
}

func CtxInfo(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Info()
}

func CtxDebug(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Debug()
}

func CtxFatal(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Fatal()
}

func CtxError(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Error()
}

func CtxWarn(ctx context.Context) *zerolog.Event {
	return zerolog.Ctx(ctx).Warn()
}

// ctx management

func CtxExtend(ctx context.Context, extend func(zerolog.Context)) context.Context {
	zctx := zerolog.Ctx(ctx).With()
	extend(zctx)
	return zctx.Logger().WithContext(ctx)
}
