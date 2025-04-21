package broadcast

import (
	"context"
	"fmt"
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

func (q *DeliveryQueue) Enqueue(ctx context.Context, to ValidatorPublicInfo, delivery Delivery) error {
	if err := q.WaitForCapacity(ctx, 1); err != nil {
		return err // context expired
	}

	q.lk.Lock()
	defer q.lk.Unlock()
	defer q.cond.Broadcast()
	channel := delivery.Msg.Header.Channel
	q.chq[channel] = append(q.chq[channel], newDeliveryTo(to, delivery))
	return nil
}

// Discard drops all queued deliveries on the given channel.
func (q *DeliveryQueue) Discard(ch Channel) {
	q.lk.Lock()
	defer q.lk.Unlock()
	defer q.cond.Broadcast()
	delete(q.chq, ch)
}

func (q *DeliveryQueue) WaitForCapacity(ctx context.Context, maxBacklog int) error {
	q.lk.Lock()
	defer q.lk.Unlock()
	for {
		n := q.deliveriesInQueue()
		// fmt.Println("n", n, "maxBacklog", maxBacklog)
		if n < maxBacklog {
			break
		}
		// fmt.Println("WAIT: n", n, "maxBacklog", maxBacklog)
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for capacity context expired")
		default:
		}
		q.cond.Wait()
	}
	return nil
}

func (q *DeliveryQueue) deliveriesInQueue() int {
	n := 0
	for _, chq := range q.chq {
		n += len(chq)
	}
	return n
}

func (q *DeliveryQueue) flush(ctx context.Context) {

	sem := make(chan struct{}, q.maxpar)

	for {
		// block until there are deliveries in the queue
		q.lk.Lock()
		deliveries := q.linearizeAndClearDeliveries()
		for len(deliveries) == 0 {
			q.cond.Wait()
			q.cond.Broadcast() // wake up WaitForCapacity
			deliveries = q.linearizeAndClearDeliveries()
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
	for channel, ds := range q.chq {
		l = append(l, ds...)
		delete(q.chq, channel)
	}
	// TODO: should we randomize the order?
	return l
}

func (q *DeliveryQueue) deliver(ctx context.Context, dto deliveryTo) {
	log.Infof("sending message %d on channel %s to %s",
		dto.Delivery.Msg.Header.Seqno,
		protoutil.ChannelIDString(dto.Delivery.Msg.Header.Channel.Channel_id),
		protoutil.ValidatorPublicShortString(dto.To),
	)
	c, err := DialBroadcastClient(ctx, q.dialer, dto.To.Address)
	if err != nil {
		log.Errorf("broadcast queue dialing %s: %v", protoutil.ValidatorPublicShortString(dto.To), err)
		return
	}
	defer c.Close()

	err = c.Broadcast_deliver(ctx, dto.Delivery)
	if err != nil {
		log.Errorf("sending message %d on channel %s to %s: %s",
			dto.Delivery.Msg.Header.Seqno,
			protoutil.ChannelIDString(dto.Delivery.Msg.Header.Channel.Channel_id),
			protoutil.ValidatorPublicShortString(dto.To),
			err,
		)
		return
	}
	q.metrics.NumSent.Add(ctx, 1)
	q.metrics.SendThroughput.Add(ctx, int64(len(dto.Delivery.Msg.Payload)))
}
