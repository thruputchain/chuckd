package scpkit

import (
	"errors"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// VerifyEnvelope verifies that env.Signature is a valid signature over the
// XDR-marshaled env.Statement under the public key contained in
// env.Statement.Node_id. It returns true iff the signature is valid.
func VerifyEnvelope(m libxdr.XdrMarshaler, env *proto.SCPEnvelope) error {
	if env.Statement.Node_id.Scheme != proto.Ed25519 {
		return errors.New("unsupported public key scheme")
	}

	if !IsSaneSCPEnvelope(env) {
		return errors.New("envelope is not sane")
	}

	pubKey := *env.Statement.Node_id.Ed25519()
	if !protoutil.VerifyXdr(m, pubKey, &env.Statement, env.Signature) {
		return errors.New("invalid signature")
	}
	return nil
}
