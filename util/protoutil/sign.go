package protoutil

import (
	"crypto/ed25519"
	"crypto/sha512"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type Signatory interface {
	Signer() proto.ValidatorPublicInfo
	SignXdr(xdrutil.XdrMarshaler, xdr.XdrType) proto.Signature
	VerifyXdr(m xdrutil.XdrMarshaler, signer proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool
}

type signer struct {
	priv proto.ValidatorPrivateInfo
}

func SignerFromPrivate(priv proto.ValidatorPrivateInfo) Signatory {
	return &signer{priv: priv}
}

func (s *signer) Signer() proto.ValidatorPublicInfo {
	return s.priv.Public_info
}

func (s *signer) SignXdr(m xdrutil.XdrMarshaler, value xdr.XdrType) proto.Signature {
	return SignXdr(m, s.priv.Secret_key, value)
}

func (s *signer) VerifyXdr(m xdrutil.XdrMarshaler, signer proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool {
	return VerifyXdr(m, s.Signer().Public_key, value, sig)
}

func SignBytes(secretKey proto.ValidatorSecretKey, msg []byte) proto.Signature {
	msgHash := sha512.Sum512(msg)
	signatureBytes := ed25519.Sign(secretKey[:], msgHash[:])
	var signature proto.Signature
	signature.Scheme = proto.Ed25519
	*signature.Ed25519() = proto.Ed25519Signature(signatureBytes)
	return signature
}

func SignXdr(m xdrutil.XdrMarshaler, secretKey proto.ValidatorSecretKey, value xdr.XdrType) proto.Signature {
	return SignBytes(secretKey, m.MarshalXdr(value))
}

func VerifyBytes(pubKey proto.ValidatorPublicKey, msg []byte, signature []byte) bool {
	msgHash := sha512.Sum512(msg)
	return ed25519.Verify(pubKey[:], msgHash[:], signature)
}

func VerifyXdr(m xdrutil.XdrMarshaler, pubKey proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool {
	if sig.Scheme != proto.Ed25519 {
		return false
	}
	sigBytes := *sig.Ed25519()
	return VerifyBytes(pubKey, m.MarshalXdr(value), sigBytes[:])
}
