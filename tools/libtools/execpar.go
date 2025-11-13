package libtools

import (
	"context"
)

func ShExecPar(
	ctx context.Context,
	script []string,
) error {

	ech := make(chan error, len(script))

	for _, s := range script {
		go func(s string) {
			_, _, err := ExecBlock(ctx, s)
			ech <- err
		}(s)
	}

	var r error
	for range script {
		err := <-ech
		if err != nil {
			r = err
		}
	}
	return r
}

func SshExecPar(
	ctx context.Context,
	host []string,
	script []string,
) error {

	ech := make(chan error, len(script))

	for i := range script {
		go func() {
			_, _, err := SshExecBlock(ctx, host[i], script[i])
			ech <- err
		}()
	}

	var r error
	for range script {
		err := <-ech
		if err != nil {
			r = err
		}
	}
	return r
}
