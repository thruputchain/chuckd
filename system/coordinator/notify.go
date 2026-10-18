package coordinator

import (
	"context"
	"sync"

	"github.com/thruputchain/chuckd/sub/baton/batonkit"
)

type Notify interface {
	Committed(ctx context.Context, cb *batonkit.CommittableBlock)
}

type IgnoreNotify struct{}

func (IgnoreNotify) Committed(ctx context.Context, cb *batonkit.CommittableBlock) {}

type NotifyReceive struct {
	committed chan *batonkit.CommittableBlock
	stopped   chan struct{}
	stopOnce  sync.Once
}

func NewNotifyReceive() *NotifyReceive {
	return &NotifyReceive{
		committed: make(chan *batonkit.CommittableBlock),
		stopped:   make(chan struct{}),
	}
}

// Stop releases blocked Committed calls and prevents future Committed calls from blocking.
func (x *NotifyReceive) Stop() {
	x.stopOnce.Do(func() {
		close(x.stopped)
	})
}

func (x *NotifyReceive) Committed(ctx context.Context, cb *batonkit.CommittableBlock) {
	select {
	case <-ctx.Done():
	case <-x.stopped:
	case x.committed <- cb:
	}
}

func (x *NotifyReceive) ReceiveCommitted(ctx context.Context) *batonkit.CommittableBlock {
	select {
	case <-ctx.Done():
		return nil
	case <-x.stopped:
		return nil
	case cb := <-x.committed:
		return cb
	}
}

// ReceiveCommittedChannel returns the commit stream. Callers must provide their own
// cancellation; Stop does not close this channel.
func (x *NotifyReceive) ReceiveCommittedChannel() <-chan *batonkit.CommittableBlock {
	return x.committed
}
