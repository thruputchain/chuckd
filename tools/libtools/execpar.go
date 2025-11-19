package libtools

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
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
	after []string,
	env []map[string]string,
) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ech := make(chan error, len(script))

	for i := range script {
		go func() {
			_, _, errScript := SshExecBlock(ctx, host[i], script[i], env[i])
			if errScript != nil {
				liblog.Errorf("script on %s failed (%v)", host[i], errScript)
			}
			if after[i] != "" {
				if errScript == nil {
					env[i]["HOG_SCRIPT_EXIT"] = "0"
				} else {
					env[i]["HOG_SCRIPT_EXIT"] = "255"
				}
				_, _, errAfter := SshExecBlock(ctx, host[i], after[i], env[i])
				if errAfter != nil {
					liblog.Errorf("after-script on %s failed (%v)", host[i], errAfter)
				}
			}
			ech <- errScript
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
