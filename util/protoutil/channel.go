package protoutil

import proto "github.com/groundhogchain/protocol-go"

func ChannelIDString(id proto.BroadcastChannelID) string {
	return "cid:" + EncodeBase64(id[:])
}

func ChannelString(c proto.BroadcastChannel) string {
	return "(" + ValidatorIDString(c.Source_id) + " " + ChannelIDString(c.Channel_id) + ")"
}
