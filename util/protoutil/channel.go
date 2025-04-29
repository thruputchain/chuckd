package protoutil

import (
	"encoding/hex"

	proto "github.com/groundhogchain/protocol-go"
)

func ChannelIDString(id proto.BroadcastChannelID) string {
	return "cid:" + hex.EncodeToString(id[:ShortHashLen])
}

func ChannelString(c proto.BroadcastChannel) string {
	return "(" + ValidatorIDString(c.Source_id) + " " + ChannelIDString(c.Channel_id) + ")"
}

func ChannelIsStreaming(c proto.BroadcastChannel) bool {
	return c.Batch_count < 0
}
