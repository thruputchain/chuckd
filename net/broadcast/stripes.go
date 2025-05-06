package broadcast

import (
	"context"
	"sync"
)

type StripedBroadcast struct {
	b       StreamingBroadcast
	stripes []SendChannel
	lk      sync.Mutex
	next    int // next stripe
}

func MakeStripedSender(ctx context.Context, b StreamingBroadcast, nstripes int) *StripedBroadcast {
	x := &StripedBroadcast{
		b:       b,
		stripes: make([]SendChannel, nstripes),
	}
	for i := 0; i < nstripes; i++ {
		x.stripes[i] = b.MakeChannel(ctx)
	}
	return x
}

func (x *StripedBroadcast) Send(ctx context.Context, payload []byte) {
	x.lk.Lock()
	n := x.next
	x.next += 1
	x.lk.Unlock()
	x.stripes[n%len(x.stripes)].Send(ctx, payload)
}

func (x *StripedBroadcast) Close() {
	for _, s := range x.stripes {
		s.Close()
	}
}
