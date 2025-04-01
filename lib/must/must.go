package must

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/log"
)

func Panic(ctx context.Context, err error) {
	panic(mkErr(ctx, err))
}

func Errorf(ctx context.Context, format string, args ...any) {
	Panic(ctx, fmt.Errorf(format, args...))
}

func Assert(ctx context.Context, cond bool, err error) {
	if cond {
		return
	}
	Panic(ctx, err)
}

func Assertf(ctx context.Context, cond bool, format string, args ...any) {
	if cond {
		return
	}
	Errorf(ctx, format, args...)
}

func NoError(ctx context.Context, err error) {
	if err != nil {
		Panic(ctx, err)
	}
}

func Try(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(*Error).Unwrap()
		}
	}()
	f()
	return nil
}

func TryThru(f func()) (err *Error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(*Error)
		}
	}()
	f()
	return nil
}

func TryWithStack(f func()) (err error, stack []byte) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(*Error).Unwrap()
			stack = r.(*Error).Stack
		}
	}()
	f()
	return nil, nil
}

func Try1[R1 any](f func() R1) (_ R1, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(*Error).Unwrap()
		}
	}()
	return f(), nil
}

func Try1Thru[R1 any](f func() R1) (_ R1, err *Error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(*Error)
		}
	}()
	return f(), nil
}

func LogPanic(r any) {
	if xerr, ok := r.(*Error); ok {
		if xerr != nil {
			if log.IsVerbose() {
				log.Error().Str("stack", xerr.StackString()).Msg(xerr.Error())
			} else {
				log.Error().Msg(xerr.Error())
			}
		} else {
			log.Errorf("encountered panic without error payload")
		}
	} else {
		log.Error().Msg(xerr.Error())
	}
}
