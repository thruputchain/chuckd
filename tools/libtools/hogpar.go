package libtools

import (
	"context"
)

func HogPar(
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
