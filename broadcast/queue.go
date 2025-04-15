package broadcast

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
)

type DeliveryQueue struct {
	cond *sync.Cond
	lk   sync.Mutex
	chq  map[Channel][]deliveryTo
}

type deliveryTo struct {
	To       ValidatorPublicInfo
	Delivery Delivery
}

func newDeliveryTo(to ValidatorPublicInfo, delivery Delivery) deliveryTo {
	return deliveryTo{To: to, Delivery: delivery}
}

func NewDeliveryQueue(ctx context.Context) *DeliveryQueue {
	q := &DeliveryQueue{
		chq: map[Channel][]deliveryTo{},
	}
	q.cond = sync.NewCond(&q.lk)
	go q.flush(ctx)
	return q
}

func (q *DeliveryQueue) Enqueue(to ValidatorPublicInfo, delivery Delivery) {
	q.lk.Lock()
	defer q.lk.Unlock()
	defer q.cond.Broadcast()
	q.chq[delivery.Msg.Header.Channel] = append(
		q.chq[delivery.Msg.Header.Channel],
		newDeliveryTo(to, delivery),
	)
}

// Discard drops all queued deliveries on the given channel.
func (q *DeliveryQueue) Discard(ch Channel) {
	q.lk.Lock()
	defer q.lk.Unlock()
	defer q.cond.Broadcast()
	delete(q.chq, ch)
}

const DELIVERY_CONCURRENCY = 100

func (q *DeliveryQueue) flush(ctx context.Context) {

	sem := make(chan struct{}, DELIVERY_CONCURRENCY)

	for {
		// block until there are deliveries in the queue
		q.lk.Lock()
		deliveries := q.linearizeAndClearDeliveries()
		for len(deliveries) == 0 {
			q.cond.Wait()
		}
		q.lk.Unlock()

		// deliver in parallel with a throttle
		for _, dto := range deliveries {
			sem <- struct{}{} // acquire a delivery slot
			go func(dto deliveryTo) {
				defer func() { <-sem }() // release delivery slot
				q.deliver(ctx, dto)
			}(dto)
		}
	}
}

func (q *DeliveryQueue) linearizeAndClearDeliveries() []deliveryTo {
	l := []deliveryTo{}
	for key, ds := range q.chq {
		l = append(l, ds...)
		delete(q.chq, key)
	}
	// TODO: should we randomize the order?
	return l
}

func (q *DeliveryQueue) deliver(ctx context.Context, dto deliveryTo) {
	log.Infof("delivering message #%d on channel id %x to %s",
		dto.Delivery.Msg.Header.Seqno,
		dto.Delivery.Msg.Header.Channel.Channel_id,
		protoutil.ValidatorPublicShortString(dto.To),
	)
	c := NewBroadcastClient(ctx, dto.To.Address)
	defer c.Close()

	err := xdrutil.CallRpcVoid(c.Broadcast_deliver, dto.Delivery)
	if err != nil {
		log.Errorf("delivering message #%d on channel id %x to %s: %s",
			dto.Delivery.Msg.Header.Seqno,
			dto.Delivery.Msg.Header.Channel.Channel_id,
			protoutil.ValidatorPublicShortString(dto.To),
			err,
		)
		return
	}
}
