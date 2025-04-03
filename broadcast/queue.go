package broadcast

import (
	"context"
	"sync"

	proto "github.com/groundhogchain/protocol-go"
)

type DeliveryQueue struct {
	lk sync.Mutex
	ch map[Channel][]proto.Delivery
}

func NewDeliveryQueue(ctx context.Context) *DeliveryQueue {
	q := &DeliveryQueue{
		ch: map[Channel][]proto.Delivery{},
	}
	go q.flush(ctx)
	return q
}

func (q *DeliveryQueue) Enqueue(v proto.ValidatorPublicInfo, delivery proto.Delivery) {
	q.lk.Lock()
	defer q.lk.Unlock()
	q.ch[delivery.Msg.Header.Channel] = append(q.ch[delivery.Msg.Header.Channel], delivery)
}

// Discard drops all queued deliveries on the given channel.
func (q *DeliveryQueue) Discard(ch Channel) {
	panic("not implemented")
}

func (q *DeliveryQueue) flush(ctx context.Context) {
	panic("XXX")
}
