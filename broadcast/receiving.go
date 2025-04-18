package broadcast

import (
	"context"
	"fmt"
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
)

type receiving struct {
	svc       *BroadcastService
	channel   Channel
	treeNodes MulticastTreeNodes
	children  []ValidatorPublicInfo
	//
	lk        sync.Mutex
	nextSeqNo int64
	received  []Message
	deliverCh chan Message
}

func newReceiving(
	ctx context.Context,
	svc *BroadcastService,
	channel Channel,
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
	deliveries []Delivery,
) *receiving {
	x := &receiving{
		svc:       svc,
		channel:   channel,
		treeNodes: treeNodes,
		children:  children,
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

// called inside a service lock!
func (x *receiving) deliver(ctx context.Context, delivery Delivery) {
	msg := delivery.Msg
	defer func() { x.deliverCh <- msg }()
	defer x.relay(ctx, delivery)

	x.lk.Lock()
	defer x.lk.Unlock()

	x.received = append(x.received, msg)

	x.svc.metrics.NumReceived.Add(ctx, 1)
	x.svc.metrics.NumNotReceived.Add(ctx, int64(msg.Header.Seqno)-x.nextSeqNo)
	x.nextSeqNo = int64(msg.Header.Seqno) + 1
	x.svc.metrics.ReceiveThroughput.Add(ctx, int64(len(delivery.Msg.Payload)))
}

func (x *receiving) relay(ctx context.Context, delivery Delivery) {
	for _, v := range x.children {
		reDelivery := Delivery{
			Relayer:    x.svc.ValidatorPublicInfo(),
			Msg:        delivery.Msg,
			Tree_nodes: delivery.Tree_nodes,
		}
		x.svc.enqueueDelivery(ctx, v, reDelivery)
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
