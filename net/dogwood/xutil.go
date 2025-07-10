package dogwood

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/magnolia"
	"github.com/klauspost/reedsolomon"
)

type MagnoliaEncoderPool struct {
	pool sync.Pool
}

func NewMagnoliaEncoderPool(
	ctx context.Context,
	nPaySheaves int,
	nParitySheaves int,
	maxBytesPerChunk int,
) *MagnoliaEncoderPool {
	x := &MagnoliaEncoderPool{}
	x.pool.New = func() any {
		return magnolia.NewEncoder(
			ctx,
			nPaySheaves,
			nParitySheaves,
			maxBytesPerChunk,
		)
	}
	return x
}

func (x *MagnoliaEncoderPool) Get() *magnolia.Encoder {
	return x.pool.Get().(*magnolia.Encoder)
}

func (x *MagnoliaEncoderPool) Recycle(v *magnolia.Encoder) {
	x.pool.Put(v)
}

type MagnoliaDecoderPool struct {
	pool sync.Pool
}

func NewMagnoliaDecoderPool(
	ctx context.Context,
	nPaySheaves int,
	nParitySheaves int,
	maxBytesPerChunk int,
) *MagnoliaDecoderPool {
	x := &MagnoliaDecoderPool{}
	x.pool.New = func() any {
		return magnolia.NewDecoder(
			ctx,
			nPaySheaves,
			nParitySheaves,
			maxBytesPerChunk,
		)
	}
	return x
}

func (x *MagnoliaDecoderPool) Get() *magnolia.Decoder {
	return x.pool.Get().(*magnolia.Decoder)
}

func (x *MagnoliaDecoderPool) Recycle(v *magnolia.Decoder) {
	x.pool.Put(v)
}

type RSEncoderPool struct {
	pool sync.Pool
}

func NewRSEncoderPool(ctx context.Context, numPayload int, numParity int) *RSEncoderPool {
	x := &RSEncoderPool{}
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

func (x *RSEncoderPool) Get() reedsolomon.Encoder {
	return x.pool.Get().(reedsolomon.Encoder)
}

func (x *RSEncoderPool) Recycle(v reedsolomon.Encoder) {
	x.pool.Put(v)
}
