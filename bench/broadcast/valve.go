package benchbroadcast

import (
	"context"
	"time"
)

func Flood(ctx context.Context, numTicks int, fn func(int)) {
	for i := 0; i < numTicks; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			fn(i)
		}
	}
}

func Drip(ctx context.Context, ticksPerSec int, numTicks int, fn func(int)) {

	t := time.NewTicker(time.Second / time.Duration(ticksPerSec))
	n := 0
	for n < numTicks {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn(n)
			n += 1
		}
	}
}
