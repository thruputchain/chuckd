package protoutil

import (
	"crypto/ed25519"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

func SignBytes(secretKey proto.ValidatorSecretKey, msg []byte) proto.Signature {
	signatureBytes := ed25519.Sign(secretKey[:], msg)
	var signature proto.Signature
	signature.Scheme = proto.Ed25519
	*signature.Ed25519() = proto.Ed25519Signature(signatureBytes)
	return signature
}

func SignXdr(secretKey proto.ValidatorSecretKey, value xdr.XdrType) proto.Signature {
	return SignBytes(secretKey, xdrutil.MarshalXdr(value))
}

func VerifyBytes(pubKey proto.ValidatorPublicKey, msg []byte, signature []byte) bool {
	return ed25519.Verify(pubKey[:], msg, signature)
}

func VerifyXdr(pubKey proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool {
	if sig.Scheme != proto.Ed25519 {
		return false
	}
	sigBytes := *sig.Ed25519()
	return VerifyBytes(pubKey, xdrutil.MarshalXdr(value), sigBytes[:])
}
