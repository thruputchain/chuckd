package broadcast

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
)

type sending struct {
	svc       *BroadcastService
	channel   Channel
	treeNodes MulticastTreeNodes
	children  []ValidatorPublicInfo
	streams   []*stream
	//
	lk   sync.Mutex
	sent []Message
}

func newSending(
	ctx context.Context,
	svc *BroadcastService,
	channel Channel,
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
) *sending {
	streams := make([]*stream, len(children))
	for i := range children {
		streams[i] = newStream(
			ctx,
			svc.metrics.DeliveryQueueMetrics(),
			svc.net,
			svc.ValidatorPublicInfo(),
			children[i],
			channel,
			2*len(children),
		)
	}
	return &sending{
		svc:       svc,
		channel:   channel,
		treeNodes: treeNodes,
		children:  children,
		streams:   streams,
	}
}

func (x *sending) Channel() Channel {
	return x.channel
}

func (x *sending) Send(ctx context.Context, payload []byte) error {

	// create broadcast message

	x.lk.Lock()

	var attachTree bool
	var prevMsgHash *MessageHash
	if len(x.sent) > 0 {
		h := protoutil.HashOfMessage(x.sent[len(x.sent)-1])
		prevMsgHash = &h
	} else {
		attachTree = true
	}

	msg := Message{
		Header: Header{
			Source:           x.svc.ValidatorPublicInfo(),
			Source_signature: protoutil.SignXdr(x.svc.SecretKey(), &x.channel),
			Channel:          x.channel,
			Seqno:            SeqNo(len(x.sent)),
			Prev_msg:         prevMsgHash,
		},
		Payload: payload,
	}
	x.sent = append(x.sent, msg)

	x.lk.Unlock()

	// prepare delivery structure

	var tree *MulticastTreeNodes
	if attachTree {
		tree = &x.treeNodes
	}
	delivery := Delivery{
		Relayer:    x.svc.ValidatorPublicInfo(),
		Msg:        msg,
		Tree_nodes: tree,
	}

	// send message to delivery queue

	for _, v := range x.streams {
		v.Enqueue(ctx, delivery)
	}

	return nil
}

func (x *sending) Close() {

	defer func() {
		for i := range x.streams {
			x.streams[i].Close()
		}
	}()

	x.svc.lk.Lock()
	defer x.svc.lk.Unlock()

	delete(x.svc.sending, x.channel)
	x.svc.trees.Deref(x.channel.Tree_hash)
}

func (x *sending) reconcile(msgHashes []MessageHash) []Message {
	x.lk.Lock()
	defer x.lk.Unlock()
	return protoutil.FindMessages(msgHashes, x.sent)
}
