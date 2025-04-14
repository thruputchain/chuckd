package broadcast

import "sync"

type receiving struct {
	svc       *BroadcastService
	channel   Channel
	treeNodes MulticastTreeNodes
	children  []ValidatorPublicInfo
	//
	lk        sync.Mutex
	received  []Message
	deliverCh chan Message
}

func newReceiving(
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
		x.deliver(delivery)
	}
	return x
}

func (x *receiving) Channel() Channel {
	return x.channel
}

// called inside a service lock!
func (x *receiving) deliver(delivery Delivery) {
	msg := delivery.Msg
	defer func() { x.deliverCh <- msg }()
	defer x.relay(delivery)
	x.lk.Lock()
	defer x.lk.Unlock()
	x.received = append(x.received, msg)
}

func (x *receiving) relay(delivery Delivery) {
	for _, v := range x.children {
		x.svc.enqueueDelivery(v, delivery)
	}
}

func (x *receiving) Receive() (seqno SeqNo, payload []byte) {
	msg := <-x.deliverCh
	return msg.Header.Seqno, msg.Payload
}

func (x *receiving) Close() {
	x.svc.lk.Lock()
	defer x.svc.lk.Unlock()

	delete(x.svc.receiving, x.channel)
	x.svc.trees.Deref(x.channel.Tree_hash)
}
