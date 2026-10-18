package protoutil

import (
	"crypto"
	"crypto/ed25519"
	"crypto/sha512"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type Signatory interface {
	Signer() proto.ValidatorPublicInfo
	SignXdr(libxdr.XdrMarshaler, xdr.XdrType) proto.Signature
	VerifyXdr(m libxdr.XdrMarshaler, value xdr.XdrType, sig proto.Signature) bool
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

func (s *signer) SignXdr(m libxdr.XdrMarshaler, value xdr.XdrType) proto.Signature {
	return SignXdr(m, s.priv.Secret_key, value)
}

func (s *signer) VerifyXdr(m libxdr.XdrMarshaler, value xdr.XdrType, sig proto.Signature) bool {
	return VerifyXdr(m, s.Signer().Public_key, value, sig)
}

func SignBytes(secretKey proto.ValidatorSecretKey, msg []byte) proto.Signature {

	msgHash := sha512.Sum512(msg)
	privKey := ed25519.PrivateKey(secretKey.Ed25519()[:])
	signatureBytes, err := privKey.Sign(nil, msgHash[:], crypto.SHA512)
	if err != nil {
		panic(err)
	}

	var signature proto.Signature
	signature.Scheme = proto.Ed25519
	*signature.Ed25519() = proto.Ed25519Signature(signatureBytes)
	return signature
}

func SignXdr(m libxdr.XdrMarshaler, secretKey proto.ValidatorSecretKey, value xdr.XdrType) proto.Signature {
	bb := m.MarshalXdr(value)
	defer bb.Recycle()
	return SignBytes(secretKey, bb.Bytes())
}

func VerifyBytes(pubKey proto.ValidatorPublicKey, msg []byte, signature []byte) bool {
	if pubKey.Scheme != proto.Ed25519 {
		return false
	}

	msgHash := sha512.Sum512(msg)
	err := ed25519.VerifyWithOptions(pubKey.Ed25519()[:], msgHash[:], signature, &ed25519.Options{Hash: crypto.SHA512})
	return err == nil
}

func VerifyXdr(m libxdr.XdrMarshaler, pubKey proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool {
	if pubKey.Scheme != proto.Ed25519 || sig.Scheme != proto.Ed25519 {
		return false
	}
	sigBytes := *sig.Ed25519()
	bb := m.MarshalXdr(value)
	defer bb.Recycle()
	return VerifyBytes(pubKey, bb.Bytes(), sigBytes[:])
}
