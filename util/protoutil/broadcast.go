package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func HashOfMessage(msg proto.BroadcastMessage) proto.BroadcastMessageHash {
	return sha256.Sum256(xdrutil.MarshalXdr(&msg))
}

func VerifyDelivery(delivery proto.Delivery) bool {

	if !VerifyXdr(
		delivery.Msg.Header.Source.Public_key,
		&delivery.Msg.Header.Channel,
		delivery.Msg.Header.Source_signature,
	) {
		return false
	}

	if !ValidatorIDEqual(
		ValidatorIDFromPublic(delivery.Msg.Header.Source),
		delivery.Msg.Header.Channel.Source_id,
	) {
		return false
	}

	return true
}

func FindMessages(msgHashes []proto.BroadcastMessageHash, msgs []proto.BroadcastMessage) []proto.BroadcastMessage {

	hs := map[proto.BroadcastMessageHash]bool{}
	for _, h := range msgHashes {
		hs[h] = true
	}

	found := []proto.BroadcastMessage{}
	for _, msg := range msgs {
		if hs[HashOfMessage(msg)] {
			found = append(found, msg)
		}
	}

	return found
}
