package coordinator

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
)

type Notify interface {
	Committed(ctx context.Context, cb *batonkit.CommittableBlock)
}

type IgnoreNotify struct{}

func (IgnoreNotify) Committed(ctx context.Context, cb *batonkit.CommittableBlock) {}

type NotifyReceive struct {
	committed chan *batonkit.CommittableBlock
}

func NewNotifyReceive() *NotifyReceive {
	return &NotifyReceive{
		committed: make(chan *batonkit.CommittableBlock),
	}
}

func (x *NotifyReceive) Committed(ctx context.Context, cb *batonkit.CommittableBlock) {
	x.committed <- cb
}

func (x *NotifyReceive) ReceiveCommitted(ctx context.Context) *batonkit.CommittableBlock {
	return <-x.committed
}

func (x *NotifyReceive) ReceiveCommittedChannel() <-chan *batonkit.CommittableBlock {
	return x.committed
}
