package broadcast

import (
	"context"
	"fmt"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libmetrics"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

type receiving struct {
	svc       *BroadcastService
	channel   Channel
	treeNodes MulticastTreeNodes
	children  []ValidatorPublicInfo
	streams   []*queue
	//
	lk        sync.Mutex
	nextSeqNo int64
	received  []Message
	deliverCh chan Message
}

func newReceiving(
	ctx context.Context, // must be a long-lived context
	svc *BroadcastService,
	channel Channel,
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
	deliveries []Delivery,
) *receiving {
	streams := make([]*queue, len(children))
	for i := range children {
		streams[i] = newQueue(
			ctx,
			svc.metrics.DeliveryQueueMetrics(),
			svc.net,
			svc.ValidatorPublicInfo(),
			children[i],
			channel,
			2*len(children),
		)
	}
	x := &receiving{
		svc:       svc,
		channel:   channel,
		treeNodes: treeNodes,
		children:  children,
		streams:   streams,
		//
		received:  nil,
		deliverCh: make(chan Message, len(deliveries)),
	}
	for _, delivery := range deliveries {
		x.deliver(ctx, delivery)
	}
	return x
}

func (x *receiving) Channel() Channel {
	return x.channel
}

func (x *receiving) deliver(ctx context.Context, delivery Delivery) {
	msg := delivery.Msg

	defer func() { x.deliverCh <- msg }()

	defer x.relay(ctx, delivery)

	x.lk.Lock()
	defer x.lk.Unlock()

	x.received = append(x.received, msg)

	x.svc.metrics.NumReceived.Add(ctx, 1, libmetrics.WithCtxAttributes(ctx))
	x.svc.metrics.NumNotReceived.Add(ctx, int64(msg.Header.Seqno)-x.nextSeqNo, libmetrics.WithCtxAttributes(ctx))
	x.nextSeqNo = int64(msg.Header.Seqno) + 1
	x.svc.metrics.ReceiveThroughput.Add(ctx, int64(len(delivery.Msg.Payload)), libmetrics.WithCtxAttributes(ctx))
}

func (x *receiving) relay(ctx context.Context, delivery Delivery) {
	for _, v := range x.streams {
		reDelivery := Delivery{
			Relayer:    x.svc.ValidatorPublicInfo(),
			Msg:        delivery.Msg,
			Tree_nodes: delivery.Tree_nodes,
		}
		v.Enqueue(ctx, reDelivery)
	}
}

func (x *receiving) Receive(ctx context.Context) (seqno SeqNo, msgHash MessageHash, payload []byte, _ error) {
	select {
	case <-ctx.Done():
		return 0, MessageHash{}, nil, fmt.Errorf("broadcast channel receive context expired")
	case msg := <-x.deliverCh:
		return msg.Header.Seqno, protoutil.HashOfMessage(msg), msg.Payload, nil
	}
}

func (x *receiving) Close() {

	defer func() {
		for i := range x.streams {
			x.streams[i].Close()
		}
	}()

	x.svc.lk.Lock()
	defer x.svc.lk.Unlock()

	delete(x.svc.receiving, x.channel)
	x.svc.trees.Deref(x.channel.Tree_hash)
}

func (x *receiving) reconcile(msgHashes []MessageHash) []Message {
	x.lk.Lock()
	defer x.lk.Unlock()
	return protoutil.FindMessages(msgHashes, x.received)
}
