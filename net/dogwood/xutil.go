package dogwood

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/klauspost/reedsolomon"
)

type EncoderPool struct {
	pool sync.Pool
}

func NewEncoderPool(ctx context.Context, numPayload int, numParity int) *EncoderPool {
	x := &EncoderPool{}
	x.pool.New = func() any {
		enc, err := reedsolomon.New(
			int(numPayload),
			int(numParity),
		)
		must.NoError(ctx, err)
		return enc
	}
	return x
}

func (x *EncoderPool) Get() reedsolomon.Encoder {
	return x.pool.Get().(reedsolomon.Encoder)
}

func (x *EncoderPool) Recycle(v reedsolomon.Encoder) {
	x.pool.Put(v)
}
