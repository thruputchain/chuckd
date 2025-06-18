package dogwood

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Navigator struct {
	dialer  DogwoodDialer
	ttl     time.Duration
	lk      sync.Mutex
	reserve map[proto.DogwoodTreeHash]*dogwoodutil.LinkedTreeNode
	cached  map[proto.DogwoodTreeHash]*dogwoodutil.LinkedTreeNode
}

func NewNavigator(
	ctx context.Context,
	dialer DogwoodDialer,
	ttl time.Duration, // tree ttl
) *Navigator {
	return &Navigator{
		dialer: dialer,
		ttl:    ttl,
	}
}

func (nav *Navigator) Start(ctx context.Context) {
	go nav.refill(ctx)
	go nav.expire(ctx)
}

func (nav *Navigator) refill(ctx context.Context) {
	panic("XXX")
}

func (nav *Navigator) expire(ctx context.Context) {
	panic("XXX")
}

// func (nav *Navigator) GetRoutes(n int) XXX {
// 	XXX
// }
