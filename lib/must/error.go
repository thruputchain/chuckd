package must

import (
	"context"
	"runtime/debug"
)

type Error struct {
	Ctx     context.Context
	Stack   []byte
	wrapped error
}

func (x *Error) Error() string {
	return x.Unwrap().Error()
}

func (x *Error) StackString() string {
	return string(x.Stack)
}

func (x *Error) Unwrap() error {
	if x == nil {
		return nil
	}
	return x.wrapped
}

type ErrorString struct {
	Msg string `json:"msg"`
}

func (x ErrorString) Error() string {
	return x.Msg
}

func mkErr(ctx context.Context, wrapped error) *Error {
	return &Error{
		Ctx:     ctx,
		Stack:   debug.Stack(),
		wrapped: wrapped,
	}
}
