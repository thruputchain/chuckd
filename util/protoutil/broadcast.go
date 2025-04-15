package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func MulticastTreeNodeHash(treeNode proto.MulticastTreeNode) proto.MulticastTreeHash {
	return sha256.Sum256(xdrutil.MarshalXdr(&treeNode))
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
