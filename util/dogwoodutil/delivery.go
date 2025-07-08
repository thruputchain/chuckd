package dogwoodutil

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func VerifyDelivery(
	m xdrutil.XdrMarshaler,
	delivery proto.DogwoodDeliveryEnvelope,
	signerKey proto.ValidatorPublicKey,
) bool {

	return protoutil.VerifyXdr(
		m,
		signerKey,
		&delivery.Delivery,
		delivery.Signature,
	)
}
