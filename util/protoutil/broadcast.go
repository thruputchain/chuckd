package protoutil

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func HashOfMessage(m xdrutil.XdrMarshaler, msg proto.BroadcastMessage) proto.BroadcastMessageHash {
	return sha256.Sum256(m.MarshalXdr(&msg))
}

func VerifyDelivery(m xdrutil.XdrMarshaler, delivery proto.Delivery) bool {

	if !VerifyXdr(
		m,
		delivery.Msg.Header.Source.Public_key,
		&delivery.Msg.Header.Channel,
		delivery.Msg.Header.Source_signature,
	) {
		return false
	}

	if !ValidatorIDEqual(
		VIDOfPublic(delivery.Msg.Header.Source),
		delivery.Msg.Header.Channel.Source_id,
	) {
		return false
	}

	return true
}

func FindMessages(
	m xdrutil.XdrMarshaler,
	msgHashes []proto.BroadcastMessageHash,
	msgs []proto.BroadcastMessage,
) []proto.BroadcastMessage {

	hs := map[proto.BroadcastMessageHash]bool{}
	for _, h := range msgHashes {
		hs[h] = true
	}

	found := []proto.BroadcastMessage{}
	for _, msg := range msgs {
		if hs[HashOfMessage(m, msg)] {
			found = append(found, msg)
		}
	}

	return found
}

// ChannelIDInt64 is intended for testing.
func ChannelIDInt64(i int64) proto.BroadcastChannelID {
	return int64ToBytes32(i)
}

// int64ToBytes32 converts an int64 value to a [32]byte array.
// The int64 is placed at the end of the array (big-endian).
func int64ToBytes32(value int64) [32]byte {
	var result [32]byte

	// Convert int64 to 8 bytes (big-endian)
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, uint64(value))

	// Copy the 8 bytes to the end of the 32-byte array
	// This places the int64 value at the rightmost position of the array
	copy(result[32-8:], bytes)

	return result
}
