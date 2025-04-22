package broadcast

import (
	"context"
	"fmt"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type DeliveryQueue struct {
	dialer libnet.AddressDialer
	maxpar int
	//
	cond    *sync.Cond
	lk      sync.Mutex
	pending map[stream][]envelope
	//
	metrics *DeliveryQueueMetrics
}

type stream struct {
	To      proto.ValidatorID
	Channel Channel
}

func newStream(to ValidatorPublicInfo, channel Channel) stream {
	return stream{To: protoutil.ValidatorIDFromPublic(to), Channel: channel}
}

type envelope struct {
	To       ValidatorPublicInfo
	Delivery Delivery
}

func newEnvelope(to ValidatorPublicInfo, delivery Delivery) envelope {
	return envelope{To: to, Delivery: delivery}
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
		pending: map[stream][]envelope{},
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

	key := newStream(to, delivery.Msg.Header.Channel)
	q.pending[key] = append(q.pending[key], newEnvelope(to, delivery))

	return nil
}

// Discard drops all queued deliveries on the given channel.
func (q *DeliveryQueue) Discard(ch Channel) {
	q.lk.Lock()
	defer q.lk.Unlock()
	defer q.cond.Broadcast()
	for key := range q.pending {
		if key.Channel == ch {
			delete(q.pending, key)
		}
	}
}

func (q *DeliveryQueue) WaitForCapacity(ctx context.Context, maxBacklog int) error {
	q.lk.Lock()
	defer q.lk.Unlock()
	for {
		n := q.streamsInQueue()
		if n < maxBacklog {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for capacity context expired")
		default:
		}
		q.cond.Wait()
	}
	return nil
}

func (q *DeliveryQueue) streamsInQueue() int {
	return len(q.pending)
}

func (q *DeliveryQueue) flush(ctx context.Context) {

	sem := make(chan struct{}, q.maxpar)

	for {
		// block until there are deliveries in the queue
		q.cond.Broadcast() // wake up WaitForCapacity
		q.lk.Lock()
		pending := q.takePending()
		for len(pending) == 0 {
			q.cond.Wait()
			q.cond.Broadcast() // wake up WaitForCapacity
			pending = q.takePending()
		}
		q.lk.Unlock()

		// deliver in parallel with a throttle
		for _, streamDeliveries := range pending {
			sem <- struct{}{} // acquire a delivery slot
			go func(streamDeliveries []envelope) {
				defer func() { <-sem }() // release delivery slot
				q.deliver(ctx, streamDeliveries)
			}(streamDeliveries)
		}
	}
}

func (q *DeliveryQueue) takePending() (pending map[stream][]envelope) {
	pending, q.pending = q.pending, map[stream][]envelope{}
	return pending
}

func (q *DeliveryQueue) deliver(ctx context.Context, streamDeliveries []envelope) {

	if len(streamDeliveries) == 0 {
		return
	}
	to := streamDeliveries[len(streamDeliveries)-1].To

	c, err := DialBroadcastClient(ctx, q.dialer, to.Address)
	if err != nil {
		log.Errorf("broadcast queue dialing %s: %v", protoutil.ValidatorPublicShortString(to), err)
		return // all deliveries in queue lost
	}
	defer c.Close()

	for _, env := range streamDeliveries {
		log.Debugf("sending message %d on channel %s to %s",
			env.Delivery.Msg.Header.Seqno,
			protoutil.ChannelString(env.Delivery.Msg.Header.Channel),
			protoutil.ValidatorPublicShortString(env.To),
		)

		err = c.Broadcast_deliver(ctx, env.Delivery)
		if err != nil {
			log.Errorf("sending message %d on channel %s to %s: %s",
				env.Delivery.Msg.Header.Seqno,
				protoutil.ChannelString(env.Delivery.Msg.Header.Channel),
				protoutil.ValidatorPublicShortString(env.To),
				err,
			)
			return // all deliveries in queue lost
		}
		q.metrics.NumSent.Add(ctx, 1)
		q.metrics.SendThroughput.Add(ctx, int64(len(env.Delivery.Msg.Payload)))
	}
}
