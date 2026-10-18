package dogwood

import (
	"context"
	"sync"

	"github.com/thruputchain/chuckd/net/magnolia"
)

type MagnoliaEncoderPool struct {
	ctx   context.Context
	lk    sync.Mutex
	pools map[magnolia.Regime]*sync.Pool
}

func NewMagnoliaEncoderPool(ctx context.Context) *MagnoliaEncoderPool {
	return &MagnoliaEncoderPool{
		ctx:   ctx,
		pools: map[magnolia.Regime]*sync.Pool{},
	}
}

func (x *MagnoliaEncoderPool) Get(r magnolia.Regime) *magnolia.Encoder {
	x.lk.Lock()
	p, ok := x.pools[r]
	if !ok {
		p = &sync.Pool{
			New: func() any {
				return magnolia.NewEncoder(x.ctx, r)
			},
		}
		x.pools[r] = p
	}
	x.lk.Unlock()

	return p.Get().(*magnolia.Encoder)
}

func (x *MagnoliaEncoderPool) Recycle(v *magnolia.Encoder) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.pools[v.Regime()].Put(v)
}

type MagnoliaDecoderPool struct {
	ctx   context.Context
	lk    sync.Mutex
	pools map[magnolia.Regime]*sync.Pool
}

func NewMagnoliaDecoderPool(ctx context.Context) *MagnoliaDecoderPool {
	return &MagnoliaDecoderPool{
		ctx:   ctx,
		pools: map[magnolia.Regime]*sync.Pool{},
	}
}

func (x *MagnoliaDecoderPool) Get(r magnolia.Regime) *magnolia.Decoder {
	x.lk.Lock()
	p, ok := x.pools[r]
	if !ok {
		p = &sync.Pool{
			New: func() any {
				return magnolia.NewDecoder(x.ctx, r)
			},
		}
		x.pools[r] = p
	}
	x.lk.Unlock()

	return p.Get().(*magnolia.Decoder)
}

func (x *MagnoliaDecoderPool) Recycle(v *magnolia.Decoder) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.pools[v.Regime()].Put(v)
}
