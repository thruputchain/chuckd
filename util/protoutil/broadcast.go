package protoutil

import (
	proto "github.com/groundhogchain/protocol-go"
)

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
