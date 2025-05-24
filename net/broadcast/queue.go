package broadcast

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libmetrics"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

type queue struct {
	metrics  *DeliveryQueueMetrics
	dialer   libnet.AddressDialer
	from     ValidatorPublicInfo
	to       ValidatorPublicInfo
	channel  Channel
	delivery chan Delivery
}

func newQueue(
	ctx context.Context,
	metrics *DeliveryQueueMetrics,
	dialer libnet.AddressDialer,
	from ValidatorPublicInfo,
	to ValidatorPublicInfo,
	channel Channel,
	capacity int,
) *queue {
	q := &queue{
		metrics:  metrics,
		dialer:   dialer,
		from:     from,
		to:       to,
		channel:  channel,
		delivery: make(chan Delivery, capacity),
	}
	go q.loop(ctx)
	return q
}

func (q *queue) Enqueue(ctx context.Context, delivery Delivery) {
	q.delivery <- delivery
}

func (q *queue) Close() {
	close(q.delivery)
}

func (q *queue) loop(ctx context.Context) {
	var c *BroadcastClient // cached client
	defer func() {
		if c != nil {
			c.Close()
		}
	}()

	var err error
	c, err = q.dial(ctx)
	if err != nil {
		log.Errorf("broadcast queue dialing %s: %v", protoutil.VPublicShortString(q.to), err)
	}
	for {
		select {
		case <-ctx.Done():
			return // context expired
		case d, ok := <-q.delivery:
			if !ok {
				return // stream closed
			}
			if c == nil {
				log.Errorf("dropping message %d on channel %s to %s at %s: no connection",
					d.Msg.Header.Seqno,
					protoutil.ChannelString(q.channel),
					protoutil.VPublicShortString(q.to),
					protoutil.VPublicShortString(q.from))
			} else {
				if err = c.Broadcast_deliver(ctx, d); err != nil {
					log.Errorf("dropping message %d on channel %s to %s at %s: %s",
						d.Msg.Header.Seqno,
						protoutil.ChannelString(d.Msg.Header.Channel),
						protoutil.VPublicShortString(q.to),
						protoutil.VPublicShortString(q.from),
						err)
					q.metrics.NumNotSent.Add(ctx, 1, libmetrics.WithCtxAttributes(ctx))
				} else {
					q.metrics.NumSent.Add(ctx, 1, libmetrics.WithCtxAttributes(ctx))
					q.metrics.SendThroughput.Add(ctx, int64(len(d.Msg.Payload)), libmetrics.WithCtxAttributes(ctx))
				}
			}
		}
	}
}

func (q *queue) dial(ctx context.Context) (*BroadcastClient, error) {
	c, err := DialBroadcastClient(ctx, q.dialer, q.to.Address)
	if err != nil {
		return nil, err
	}
	return c, nil
}
