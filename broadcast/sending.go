package broadcast

import (
	"crypto/sha256"
	"sync"

	"github.com/groundhogchain/chuckd/util/xdrutil"
)

type sending struct {
	svc      *BroadcastService
	channel  Channel
	tree     MulticastTreeNodes
	children []ValidatorPublicInfo
	//
	lk   sync.Mutex
	sent []Message
}

func newSending(
	svc *BroadcastService,
	channel Channel,
	tree MulticastTreeNodes,
	children []ValidatorPublicInfo,
) *sending {
	return &sending{
		svc:      svc,
		channel:  channel,
		tree:     tree,
		children: children,
	}
}

func (x *sending) Channel() Channel {
	return x.channel
}

func (x *sending) Send(payload []byte) {

	// create broadcast message

	var attachTree bool
	var prevMsgHash *MessageHash
	if len(x.sent) > 0 {
		h := sha256.Sum256(xdrutil.MarshalXdr(&x.sent[len(x.sent)-1]))
		prevMsgHash = &h
	} else {
		attachTree = true
	}

	msg := Message{
		Header: Header{
			Channel:  x.channel,
			Seqno:    SeqNo(len(x.sent)),
			Prev_msg: prevMsgHash,
		},
		Payload: payload,
	}
	x.lk.Lock()
	x.sent = append(x.sent, msg)
	x.lk.Unlock()

	// prepare delivery structure

	var tree *MulticastTreeNodes
	if attachTree {
		tree = &x.tree
	}
	delivery := Delivery{
		Relayer:    x.svc.ValidatorPublicInfo(),
		Msg:        msg,
		Tree_nodes: tree,
	}

	// send message to delivery queue

	for _, v := range x.children {
		x.svc.enqueueDelivery(v, delivery)
	}
}
