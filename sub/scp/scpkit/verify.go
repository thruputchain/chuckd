package scpkit

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// VerifyEnvelope verifies that env.Signature is a valid signature over the
// XDR-marshaled env.Statement under the public key contained in
// env.Statement.Node_id. It returns true iff the signature is valid.
func VerifyEnvelope(m libxdr.XdrMarshaler, env *proto.SCPEnvelope) bool {
	if env.Statement.Node_id.Scheme != proto.Ed25519 {
		return false
	}
	pubKey := *env.Statement.Node_id.Ed25519()
	return protoutil.VerifyXdr(m, pubKey, &env.Statement, env.Signature)
}
