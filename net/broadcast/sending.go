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
	queues    []*queue
	//
	lk   sync.Mutex
	sent []Message
}

func newSending(
	_ context.Context,
	svc *BroadcastService,
	channel Channel,
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
) *sending {
	queues := make([]*queue, len(children))
	for i := range children {
		queues[i] = newQueue(
			context.Background(), // make new context because the calling ctx (from an rpc) is short-lived
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
		queues:    queues,
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

	for _, v := range x.queues {
		v.Enqueue(ctx, delivery)
	}

	return nil
}

func (x *sending) Close() {

	defer func() {
		for i := range x.queues {
			x.queues[i].Close()
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
