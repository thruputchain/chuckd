package broadcast

import (
	proto "github.com/groundhogchain/protocol-go"
)

// Epoch describes an epoch of the broadcast system.
// Each node maintains a current epoch.
// Newly created (send) channels are associated with an epoch which must be greater or equal to the current one.
// When the current epoch advances, the lifetime of channels with smaller epochs is terminated and their state is discarded.
// Similarly, incoming accepted receive-side channels are discarded after the node's current epoch progresses beyond theirs.
//
// Epochs provide a mechanism for knowing when the lifetime of a send or receive channel is over and resources can be released.
// The current epoch (of the broadcast system at a node) is controlled by the application.
// For example, in Groundhog the epoch can be derived from the round or block height of the consensus allgorithm.
type Epoch = proto.BroadcastEpoch

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

	// AdvanceEpoch advances the current epoch by delta, returning the new epoch.
	AdvanceEpoch(delta int) Epoch

	// MakeChannel creates a new channel associated with the given epoch, which must be no smaller than the current epoch.
	// The channel will be disbanded when the current epoch advances beyond the channel's epoch.
	MakeChannel(Epoch) SendChannel

	// AcceptChannel blocks until a new broadcast channel reaches this node.
	AcceptChannel() ReceiveChannel
}

type SendChannel interface {
	Channel() Channel
	// Epoch returns the epoch of this channel, which is determined by the application at this node when the channel was created.
	Epoch() Epoch
	Send(payload []byte)
}

type ReceiveChannel interface {
	Channel() Channel
	// Epoch returns the epoch of this channel, which is determined by the node that created it.
	Epoch() Epoch
	Receive() (seqno SeqNo, payload []byte, err error)
}
