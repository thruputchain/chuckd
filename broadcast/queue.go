package broadcast

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

type DeliveryQueue struct {
	dialer libnet.AddressDialer
	maxpar int
	//
	cond *sync.Cond
	lk   sync.Mutex
	chq  map[Channel][]deliveryTo
	//
	metrics *DeliveryQueueMetrics
}

type deliveryTo struct {
	To       ValidatorPublicInfo
	Delivery Delivery
}

func newDeliveryTo(to ValidatorPublicInfo, delivery Delivery) deliveryTo {
	return deliveryTo{To: to, Delivery: delivery}
}

func NewDeliveryQueue(
	ctx context.Context,
	metrics *DeliveryQueueMetrics,
	dialer libnet.AddressDialer,
	maxpar int,
) *DeliveryQueue {
	q := &DeliveryQueue{
		dialer:  dialer,
		maxpar:  maxpar,
		chq:     map[Channel][]deliveryTo{},
		metrics: metrics,
	}
	q.cond = sync.NewCond(&q.lk)
	go q.flush(ctx)
	return q
}

func (q *DeliveryQueue) Enqueue(to ValidatorPublicInfo, delivery Delivery) {
	q.lk.Lock()
	defer q.lk.Unlock()
	defer q.cond.Broadcast()
	channel := delivery.Msg.Header.Channel
	q.chq[channel] = append(q.chq[channel], newDeliveryTo(to, delivery))
}

// Discard drops all queued deliveries on the given channel.
func (q *DeliveryQueue) Discard(ch Channel) {
	q.lk.Lock()
	defer q.lk.Unlock()
	defer q.cond.Broadcast()
	delete(q.chq, ch)
}

func (q *DeliveryQueue) flush(ctx context.Context) {

	sem := make(chan struct{}, q.maxpar)

	for {
		// block until there are deliveries in the queue
		q.lk.Lock()
		deliveries := q.linearizeAndClearDeliveries()
		for len(deliveries) == 0 {
			q.cond.Wait()
			deliveries = q.linearizeAndClearDeliveries()
		}
		q.lk.Unlock()

		// deliver in parallel with a throttle
		for _, dto := range deliveries {
			log.Infof("DELIVERING to %s", protoutil.ValidatorPublicShortString(dto.To))
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
	for channel, ds := range q.chq {
		l = append(l, ds...)
		delete(q.chq, channel)
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
	c, err := DialBroadcastClient(ctx, q.dialer, dto.To.Address)
	if err != nil {
		log.Errorf("queue dialing %s: %v", protoutil.ValidatorPublicShortString(dto.To), err)
		return
	}
	defer c.Close()

	err = c.Broadcast_deliver(ctx, dto.Delivery)
	if err != nil {
		log.Errorf("delivering message #%d on channel id %x to %s: %s",
			dto.Delivery.Msg.Header.Seqno,
			dto.Delivery.Msg.Header.Channel.Channel_id,
			protoutil.ValidatorPublicShortString(dto.To),
			err,
		)
		return
	}
	q.metrics.NumSent.Add(ctx, 1)
	q.metrics.SendThroughput.Add(ctx, int64(len(dto.Delivery.Msg.Payload)))
}
