package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func testHostedBootstrapMetadata() hostedBootstrapMetadata {
	return hostedBootstrapMetadata{
		SchemaVersion:              1,
		Kind:                       "hog-hosted-bootstrap",
		NetworkID:                  "devnet-test",
		Protocol:                   proto.HogProtocol(),
		GenesisHash:                "0x" + strings.Repeat("42", 32),
		BootstrapAccountDerivation: json.RawMessage(`{"scheme":"groundhog-bootstrap-account-v1","seed":"seed"}`),
		BootstrapAccounts:          json.RawMessage(`[{"index":0,"role":"hosted_dev_account","publicKey":"0x` + strings.Repeat("11", 32) + `","walletAddress":"0x` + strings.Repeat("22", 32) + `","initialStorageTokens":"1000"}]`),
		Bootstrap:                  json.RawMessage(`{"devWallet":{"contentHash":"0x` + strings.Repeat("33", 32) + `","deployedCodeHash":"0x` + strings.Repeat("33", 32) + `","artifactHint":"dev_wallet.rv32a","deployEntrypoint":"0x` + strings.Repeat("44", 8) + `","doActionEntrypoint":"0x` + strings.Repeat("55", 8) + `","createWalletEntrypoint":"0x` + strings.Repeat("66", 8) + `"},"storageCostModel":{"hash":"0x` + strings.Repeat("77", 32) + `","baseDataItemCost":"1","bytestringCostPerByte":"2","baseContractDeploymentCost":"3","contractDeploymentCostPerByte":"4","authPerKeyCost":"5","rcqCostPerEntry":"6","replayCacheCostPerEntry":"7","balanceCost":"8"}}`),
	}
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

func TestWriteHostedRuntimeManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime-manifest.json")
	hash, err := writeHostedRuntimeManifest(testHostedBootstrapMetadata(), hostedBootstrapConfig{
		schedulerAddress:   "127.0.0.1:8070",
		indexerAddress:     "127.0.0.1:8061",
		publicRPCURL:       "http://203.0.113.10:8545",
		runtimeManifestOut: path,
	}, hostedBootstrapCommit{Height: 1, BlockHash: "0x" + strings.Repeat("88", 32)})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256.Sum256(data); got != hash {
		t.Fatal("manifest hash does not match written bytes")
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["kind"] != "hosted-devnet-runtime" {
		t.Fatalf("kind = %v", manifest["kind"])
	}
	endpoints := manifest["endpoints"].(map[string]any)
	if endpoints["scheduler"] != "127.0.0.1:8070" || endpoints["indexer"] != "127.0.0.1:8061" {
		t.Fatalf("unexpected endpoints: %+v", endpoints)
	}
	hosted := manifest["hostedBootstrap"].(map[string]any)
	if hosted["publicRpcUrl"] != "http://203.0.113.10:8545" {
		t.Fatalf("unexpected hosted extension: %+v", hosted)
	}
}

func TestWriteHostedRuntimeManifestRejectsHostnameEndpoint(t *testing.T) {
	_, err := writeHostedRuntimeManifest(testHostedBootstrapMetadata(), hostedBootstrapConfig{
		schedulerAddress:   "localhost:8070",
		runtimeManifestOut: filepath.Join(t.TempDir(), "runtime-manifest.json"),
	}, hostedBootstrapCommit{})
	if err == nil {
		t.Fatal("expected hostname scheduler endpoint to fail")
	}
}
