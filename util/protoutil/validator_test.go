package protoutil

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func TestValidatorIdentityXDRRoundTrip(t *testing.T) {
	identity, err := GenerateIdentity(rand.Reader, "127.0.0.1:9000")
	if err != nil {
		t.Fatal(err)
	}
	m := libxdr.NewXdrMarshaler()
	encoded := m.MarshalXdr(identity)
	defer encoded.Recycle()
	var decoded proto.ValidatorPrivateInfo
	if err := libxdr.UnmarshalXdrEOF(&decoded, encoded.Bytes()); err != nil {
		t.Fatal(err)
	}
	if decoded.Public_info.Public_key.Scheme != proto.Ed25519 || decoded.Secret_key.Scheme != proto.Ed25519 {
		t.Fatal("identity did not preserve Ed25519 schemes")
	}
	if !ValidatorInfoEquals(&identity.Public_info, &decoded.Public_info) {
		t.Fatal("decoded identity differs from original")
	}
	id := PublicVID(decoded.Public_info)
	if got := ValidatorToNodeID(id); *got.Ed25519() != *identity.Public_info.Public_key.Ed25519() {
		t.Fatal("validator ID did not preserve Ed25519 public key bytes")
	}
	message := []byte("validator key union round trip")
	sig := SignBytes(decoded.Secret_key, message)
	if !VerifyBytes(decoded.Public_info.Public_key, message, sig.Ed25519()[:]) {
		t.Fatal("decoded identity failed signature verification")
	}
	message[0] ^= 1
	if VerifyBytes(decoded.Public_info.Public_key, message, sig.Ed25519()[:]) {
		t.Fatal("accepted tampered message")
	}
}

func TestVerifyRejectsUnsupportedPublicKey(t *testing.T) {
	for _, scheme := range []proto.CryptographicScheme{1, -1, 123} {
		key := proto.ValidatorPublicKey{Scheme: scheme}
		if VerifyBytes(key, nil, nil) {
			t.Fatalf("VerifyBytes accepted scheme %d", scheme)
		}
		// An uninitialized marshaler ensures rejection precedes serialization.
		if VerifyXdr(libxdr.XdrMarshaler{}, key, nil, proto.Signature{}) {
			t.Fatalf("VerifyXdr accepted scheme %d", scheme)
		}
		info := proto.ValidatorPublicInfo{Public_key: key}
		if got := VPublicDisplay(info); !strings.Contains(got, "unsupported public key scheme") {
			t.Fatalf("unexpected display: %s", got)
		}
		env := proto.SCPEnvelope{Statement: proto.SCPStatement{Node_id: key}}
		if got := SCPEnvelopeDisplay(env); !strings.Contains(got, "unsupported public key scheme") {
			t.Fatalf("unexpected envelope display: %s", got)
		}
	}
}

func TestVerifyQSetRejectsUnsupportedPublicKeys(t *testing.T) {
	bad := proto.SCPQuorumSet{
		Threshold:  1,
		Validators: []proto.SCPNodeID{{Scheme: proto.CryptographicScheme(123)}},
	}
	for _, qset := range []proto.SCPQuorumSet{
		bad,
		{Threshold: 1, Inner_sets: []proto.SCPQuorumSet{{Threshold: 1, Inner_sets: []proto.SCPQuorumSet{bad}}}},
	} {
		if err := VerifyQSet(qset); err == nil || !strings.Contains(err.Error(), "unsupported public key scheme") {
			t.Fatalf("expected unsupported key error, got %v", err)
		}
	}
}
