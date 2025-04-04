package broadcast

import "sync"

type recvChannel struct {
	svc      *BroadcastService
	channel  Channel
	treeHash MulticastTreeHash
	//
	treeNodes *MulticastTreeNodes
	children  *[]ValidatorPublicInfo
	//
	lk        sync.Mutex
	received  []Message
	deliverCh chan Message
}

func newRecvChannel(
	svc *BroadcastService,
	channel Channel,
	treeHash MulticastTreeHash,
	treeNodes *MulticastTreeNodes,
) *recvChannel {
	return &recvChannel{
		svc:       svc,
		channel:   channel,
		treeHash:  treeHash,
		treeNodes: treeNodes,
		//
		received:  nil,
		deliverCh: make(chan Message, 1),
	}
}

func (x *recvChannel) Channel() Channel {
	return x.channel
}

func (x *recvChannel) Epoch() Epoch {
	return x.channel.Epoch
}

func (x *recvChannel) deliver(msg Message) {
	defer func() { x.deliverCh <- msg }()
	x.lk.Lock()
	defer x.lk.Unlock()
	x.received = append(x.received, msg)
}

func (x *recvChannel) Receive() (seqno SeqNo, payload []byte) {
	msg := <-x.deliverCh
	return msg.Header.Seqno, msg.Payload
}
