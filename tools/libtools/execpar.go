package libtools

import (
	"context"
)

func ShExecPar(
	ctx context.Context,
	script []string,
	env []map[string]string,
) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ech := make(chan error, len(script))

	for i, s := range script {
		go func(s string) {
			_, _, err := ExecBlock(ctx, s, env[i])
			ech <- err
		}(s)
	}

	for range script {
		err := <-ech
		if err != nil {
			return err
		}
	}
	return nil
}

func SshExecPar(
	ctx context.Context,
	host []string,
	script []string,
	env []map[string]string,
) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ech := make(chan error, len(script))

	for i := range script {
		go func() {
			_, _, err := SshExecBlock(ctx, host[i], script[i], env[i])
			ech <- err
		}()
	}

	for range script {
		err := <-ech
		if err != nil {
			return err
		}
	}
	return nil
}
