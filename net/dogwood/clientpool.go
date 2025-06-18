package dogwood

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	proto "github.com/groundhogchain/protocol-go"
)

type ClientPool struct {
	time   libtime.Time
	net    libnet.Network
	lookup AddressLookupFn
	ttl    time.Duration // client ttl
	lk     sync.Mutex
	client map[proto.ValidatorID]*poolClient
}

type AddressLookupFn func(proto.ValidatorID) string

// XXX: if error during client use, evict from pool
func NewClientPool(
	ctx context.Context,
	time libtime.Time,
	net libnet.Network,
	lookup AddressLookupFn,
	ttl time.Duration,
) *ClientPool {
	x := &ClientPool{
		time:   time,
		net:    net,
		lookup: lookup,
		ttl:    ttl,
		client: map[proto.ValidatorID]*poolClient{},
	}
	go x.expire(ctx)
	return x
}

// XXX: set last stamp
func (x *ClientPool) Dial(ctx context.Context, vid proto.ValidatorID) proto.Dogwood1 {
	panic("XXX")
}

func (x *ClientPool) expire(ctx context.Context) {
	for {
		x.time.Sleep(x.ttl)
		now := x.time.Now()
		x.lk.Lock()
		for key, q := range x.client {
			if now.Sub(q.GetLast()) > x.ttl {
				delete(x.client, key)
				q.Close()
			}
		}
		x.lk.Unlock()
	}
}

type poolClient struct {
	*DogwoodClient
	lk   sync.Mutex
	last time.Time
}

func (x *poolClient) SetLast(t time.Time) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.last = t
}

func (x *poolClient) GetLast() time.Time {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.last
}
