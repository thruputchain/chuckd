package dogwood

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	proto "github.com/groundhogchain/protocol-go"
)

// Navigator maintains a cache of routes rooted at this node that have been socialized with others,
// and can be used for broadcasting. It also maintains a cache of trees socialized by others.
type Navigator struct {
	// systems
	dialer DogwoodDialer
	// params
	ttl time.Duration
	cap int
	// state
	lk      sync.Mutex
	reserve map[proto.DogwoodTreeHash]*dogwoodutil.LinkedTreeNode
	cached  map[proto.DogwoodTreeHash]*dogwoodutil.LinkedTreeNode
}

func NewNavigator(
	ctx context.Context,
	dialer DogwoodDialer,
	ttl time.Duration, // tree ttl
	cap int, // min reserve capacity
) *Navigator {
	return &Navigator{
		dialer: dialer,
		ttl:    ttl,
		cap:    cap,
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
