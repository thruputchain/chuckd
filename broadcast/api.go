package broadcast

import (
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

// Broadcast describes the application-facing API for the broadcast system.
type Broadcast interface {
	// MakeChannel creates a new broadcast channel for sending.
	MakeChannel() SendChannel

	// AcceptChannel blocks until a new broadcast channel reaches this node.
	AcceptChannel() ReceiveChannel
}

type SendChannel interface {
	Channel() Channel
	Send(payload []byte)
	Close()
}

type ReceiveChannel interface {
	Channel() Channel
	Receive() (seqno SeqNo, payload []byte)
	Close()
}

type AcceptDiscriminator interface {
	IsAcceptable(channel Channel, seqno SeqNo, payload []byte) bool
}

type AcceptEverything struct{}

func (AcceptEverything) IsAcceptable(channel Channel, seqno SeqNo, payload []byte) bool {
	return true
}
