package broadcast

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type SeqNo = proto.SeqNo
type ChannelID = proto.BroadcastChannelID
type Channel = proto.BroadcastChannel
type MulticastTreeNode = proto.MulticastTreeNode
type MulticastTreeNodes = proto.MulticastTreeNodes
type MulticastTreeHash = proto.MulticastTreeHash
type ValidatorPublicInfo = proto.ValidatorPublicInfo
type ReconcileMessages = proto.ReconcileMessages
type ReconciledMessages = proto.ReconciledMessages
type Delivery = proto.Delivery
type Message = proto.BroadcastMessage
type MessageHash = proto.BroadcastMessageHash
type Header = proto.BroadcastHeader

// BatchBroadcast provides a batch broadcast abstraction, where the sender knows the entire (bulk) payload ahead of time.
type BatchBroadcast interface {
	Send(context.Context, []byte)
	Receive(context.Context) (Channel, []byte)
}

// StreamingBroadcast provides a streaming broadcast abstraction.
type StreamingBroadcast interface {
	// MakeChannel creates a new broadcast channel for sending.
	MakeChannel(context.Context) SendChannel

	// AcceptChannel blocks until a new broadcast channel reaches this node.
	AcceptChannel() ReceiveChannel
}

type SendChannel interface {
	Channel() Channel
	// errors occur when context is cancelled or times out
	Send(ctx context.Context, payload []byte) error
	Close()
}

type ReceiveChannel interface {
	Channel() Channel
	// errors occur when context is cancelled or times out
	Receive(ctx context.Context) (seqno SeqNo, msgHash MessageHash, payload []byte, err error)
	Close()
}

type AcceptDiscriminator interface {
	IsAcceptable(channel Channel, seqno SeqNo, payload []byte) bool
}

type AcceptEverything struct{}

func (AcceptEverything) IsAcceptable(channel Channel, seqno SeqNo, payload []byte) bool {
	return true
}
