package router

import proto "github.com/groundhogchain/protocol-go"

type ChannelID = proto.BroadcastChannelID
type ValidatorID = proto.ValidatorID
type MulticastTreeNodes = proto.MulticastTreeNodes
type ValidatorPublicInfo = proto.ValidatorPublicInfo
type MulticastTreeHash = proto.MulticastTreeHash

type Router interface {
	BuildTree(ValidatorID, ChannelID) (MulticastTreeNodes, []ValidatorPublicInfo, MulticastTreeHash)
}
