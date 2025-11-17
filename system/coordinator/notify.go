package coordinator

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type Notify interface {
	Committed(ctx context.Context, height proto.BlockHeight)
}

type IgnoreNotify struct{}

func (IgnoreNotify) Committed(ctx context.Context, height proto.BlockHeight) {}

type NotifyReceive struct {
	committed chan proto.BlockHeight
}

func NewNotifyReceive() *NotifyReceive {
	return &NotifyReceive{
		committed: make(chan proto.BlockHeight),
	}
}

func (x *NotifyReceive) Committed(ctx context.Context, height proto.BlockHeight) {
	x.committed <- height
}

func (x *NotifyReceive) ReceiveCommitted(ctx context.Context) proto.BlockHeight {
	return <-x.committed
}
