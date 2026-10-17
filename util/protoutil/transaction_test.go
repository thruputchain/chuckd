package protoutil

import (
	"crypto/sha256"
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// signedTxWithSignature is a fixed transaction body with one placeholder
// signature whose bytes are all sigByte.
func signedTxWithSignature(sigByte byte) proto.SignedTransaction {
	var sig proto.Signature
	sig.Scheme = proto.Ed25519
	for i := range sig.Ed25519() {
		sig.Ed25519()[i] = sigByte
	}
	return proto.SignedTransaction{
		Tx: proto.Transaction{
			Uniquifier: proto.Uniquifier{Nonce: 7},
		},
		Signatures: []proto.DecoratedSignature{{
			Public_key_id: 0,
			Signature:     sig,
		}},
	}
}

func TestHashSignedTxCoversSignedEncoding(t *testing.T) {
	tx := signedTxWithSignature(1)

	full := sha256.New()
	libxdr.MarshalXdrWriter(full, &tx)
	if got, want := HashSignedTx(&tx), proto.TransactionHash(full.Sum(nil)); got != want {
		t.Fatalf("HashSignedTx: got %x, want SHA-256 of the signed encoding %x", got, want)
	}

	body := sha256.New()
	libxdr.MarshalXdrWriter(body, &tx.Tx)
	if HashSignedTx(&tx) == proto.TransactionHash(body.Sum(nil)) {
		t.Fatal("HashSignedTx equals the hash of the body alone")
	}

	resigned := signedTxWithSignature(2)
	if HashSignedTx(&tx) == HashSignedTx(&resigned) {
		t.Fatal("HashSignedTx ignores the signatures")
	}

	withKey := signedTxWithSignature(1)
	key := proto.PublicKey{Scheme: proto.Ed25519}
	withKey.Pubkey_claim_key = &key
	if HashSignedTx(&tx) == HashSignedTx(&withKey) {
		t.Fatal("HashSignedTx ignores the pubkey claim key")
	}
}
