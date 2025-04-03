package broadcast

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type sendChannel struct {
	svc      *BroadcastService
	channel  Channel
	tree     MulticastTreeNodes
	children []proto.ValidatorPublicInfo
	sent     []proto.BroadcastMessage
}

func newSendChannel(svc *BroadcastService, channel Channel) *sendChannel {
	return &sendChannel{
		svc:      svc,
		channel:  channel,
		// tree:     XXX,
		// children: XXX,
	}
}

func (x *sendChannel) Channel() Channel {
	return x.channel
}

func (x *sendChannel) Epoch() Epoch {
	return x.channel.Epoch
}

func (x *sendChannel) Send(payload []byte) {

	// create broadcast message

	var attachTree bool
	var prevMsgHash *proto.BroadcastMessageHash
	if len(x.sent) > 0 {
		h := sha256.Sum256(xdrutil.MarshalXdr(&x.sent[len(x.sent)-1]))
		prevMsgHash = &h
	} else {
		attachTree = true
	}

	msg := proto.BroadcastMessage{
		Header: proto.BroadcastHeader{
			Channel:  x.channel,
			Seqno:    SeqNo(len(x.sent)),
			Prev_msg: prevMsgHash,
		},
		Payload: payload,
	}
	x.sent = append(x.sent, msg)

	// prepare delivery structure

	var tree *proto.MulticastTreeNodes
	if attachTree {
		tree = &x.tree
	}
	delivery := proto.Delivery{
		Relayer: x.svc.ValidatorPublicInfo(),
		Msg:     msg,
		Tree:    tree,
	}

	// send message to delivery queue

	for _, v := range x.children {
		x.svc.enqueueDelivery(v, delivery)
	}
}
