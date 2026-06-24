package cmd

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func writeHostedBootstrapMetadata(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func xdrHexSignedTransaction(t *testing.T, tx proto.SignedTransaction) string {
	t.Helper()
	var buf bytes.Buffer
	libxdr.MarshalXdrWriter(&buf, proto.XDR_SignedTransaction(&tx))
	return hex.EncodeToString(buf.Bytes())
}

func hostedBootstrapHashHex(tx proto.SignedTransaction) string {
	hash := protoutil.HashSignedTx(&tx)
	return "0x" + hex.EncodeToString(hash[:])
}

func TestLoadHostedBootstrapMetadataRejectsWrongKind(t *testing.T) {
	path := writeHostedBootstrapMetadata(t, `{
  "schemaVersion": 1,
  "kind": "wrong",
  "networkId": "devnet",
  "protocol": "`+proto.HogProtocol()+`",
  "genesisHash": "0xabc",
  "initializationTransactions": []
}`)

	if _, err := loadHostedBootstrapMetadata(path); err == nil {
		t.Fatal("expected wrong kind to fail")
	}
}

func TestDecodeHostedBootstrapTransactionsDecodesXDR(t *testing.T) {
	tx := proto.SignedTransaction{}
	txHex := xdrHexSignedTransaction(t, tx)
	metadata := hostedBootstrapMetadata{
		SchemaVersion: 1,
		Kind:          "hog-hosted-bootstrap",
		NetworkID:     "devnet",
		Protocol:      proto.HogProtocol(),
		GenesisHash:   "0xabc",
		InitializationTransactions: []struct {
			Index int    `json:"index"`
			Hash  string `json:"hash"`
			XDR   string `json:"xdr"`
		}{
			{Index: 0, Hash: hostedBootstrapHashHex(tx), XDR: txHex},
		},
	}

	txs, err := decodeHostedBootstrapTransactions(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 {
		t.Fatalf("decoded %d transactions, want 1", len(txs))
	}
}

func TestDecodeHostedBootstrapTransactionsRejectsOutOfOrderIndex(t *testing.T) {
	metadata := hostedBootstrapMetadata{
		SchemaVersion: 1,
		Kind:          "hog-hosted-bootstrap",
		NetworkID:     "devnet",
		Protocol:      proto.HogProtocol(),
		GenesisHash:   "0xabc",
		InitializationTransactions: []struct {
			Index int    `json:"index"`
			Hash  string `json:"hash"`
			XDR   string `json:"xdr"`
		}{
			{Index: 1, Hash: "0xabc", XDR: "00"},
		},
	}

	if _, err := decodeHostedBootstrapTransactions(metadata); err == nil {
		t.Fatal("expected out-of-order index to fail")
	}
}

func TestDecodeHostedBootstrapTransactionsRejectsHashMismatch(t *testing.T) {
	txHex := xdrHexSignedTransaction(t, proto.SignedTransaction{})
	metadata := hostedBootstrapMetadata{
		SchemaVersion: 1,
		Kind:          "hog-hosted-bootstrap",
		NetworkID:     "devnet",
		Protocol:      proto.HogProtocol(),
		GenesisHash:   "0xabc",
		InitializationTransactions: []struct {
			Index int    `json:"index"`
			Hash  string `json:"hash"`
			XDR   string `json:"xdr"`
		}{
			{Index: 0, Hash: "0x" + strings.Repeat("00", 32), XDR: txHex},
		},
	}

	if _, err := decodeHostedBootstrapTransactions(metadata); err == nil {
		t.Fatal("expected hash mismatch to fail")
	}
}
