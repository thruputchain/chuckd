package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
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

func writeHostedBootstrapMetadataJSON(t *testing.T, metadata hostedBootstrapMetadata) string {
	t.Helper()
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	return writeHostedBootstrapMetadata(t, string(data))
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

// testProtocolText is the protocol token as it appears in metadata JSON.
func testProtocolText() string {
	protocol := proto.HogProtocol()
	return string(protocol[:])
}

func testHostedBootstrapMetadata() hostedBootstrapMetadata {
	return hostedBootstrapMetadata{
		SchemaVersion:              1,
		Kind:                       "hog-hosted-bootstrap",
		NetworkHash:                strings.Repeat("00", 32),
		Protocol:                   testProtocolText(),
		BootstrapAccountDerivation: json.RawMessage(`{"scheme":"groundhog-bootstrap-account-v1","seed":"seed"}`),
		BootstrapAccounts:          json.RawMessage(`[{"index":0,"role":"hosted_dev_account","publicKey":"0x` + strings.Repeat("11", 32) + `","walletAddress":"0x` + strings.Repeat("22", 32) + `","initialStorageTokens":"1000"}]`),
		Bootstrap:                  json.RawMessage(`{"devWallet":{"contentHash":"0x` + strings.Repeat("33", 32) + `","deployedCodeHash":"0x` + strings.Repeat("33", 32) + `","artifactHint":"dev_wallet.rv32a","deployEntrypoint":"0x` + strings.Repeat("44", 8) + `","doActionEntrypoint":"0x` + strings.Repeat("55", 8) + `","createWalletEntrypoint":"0x` + strings.Repeat("66", 8) + `"},"storageCostModel":{"hash":"0x` + strings.Repeat("77", 32) + `","baseDataItemCost":"1","bytestringCostPerByte":"2","baseContractDeploymentCost":"3","contractDeploymentCostPerByte":"4","authPerKeyCost":"5","rcqCostPerEntry":"6","replayCacheCostPerEntry":"7","balanceCost":"8"}}`),
	}
}

func testHostedBootstrapMetadataWithTransactions(t *testing.T, txs ...proto.SignedTransaction) hostedBootstrapMetadata {
	t.Helper()
	metadata := testHostedBootstrapMetadata()
	for i, tx := range txs {
		metadata.InitializationTransactions = append(metadata.InitializationTransactions, struct {
			Index int    `json:"index"`
			Hash  string `json:"hash"`
			XDR   string `json:"xdr"`
		}{
			Index: i,
			Hash:  hostedBootstrapHashHex(tx),
			XDR:   xdrHexSignedTransaction(t, tx),
		})
	}
	return metadata
}

type fakeHostedBootstrapCoordinator struct {
	tip       proto.SchedulerGetTipResult
	result    proto.GenesisCeremonyResult
	request   *proto.GenesisCeremonyRequest
	commitErr error
	signaled  bool
}

func (x *fakeHostedBootstrapCoordinator) GetTip(context.Context) (proto.SchedulerGetTipResult, error) {
	return x.tip, nil
}

func (x *fakeHostedBootstrapCoordinator) CommitGenesisCeremony(_ context.Context, request proto.GenesisCeremonyRequest) (proto.GenesisCeremonyResult, error) {
	x.request = &request
	return x.result, x.commitErr
}

func (x *fakeHostedBootstrapCoordinator) SignalGenesisIsReady() {
	x.signaled = true
}

func testHostedBootstrapCeremonyResult(t *testing.T, height proto.BlockHeight) proto.GenesisCeremonyResult {
	t.Helper()
	result := proto.GenesisCeremonyResult{Outcome: proto.GenesisCeremonyCommitted}
	result.Commit().Db_tip.Value.Proposal_header.Block_height = height
	return result
}

func setHostedBootstrapCommitIdentity(t *testing.T, metadata hostedBootstrapMetadata, tx proto.SignedTransaction, tip *proto.DbTip) {
	t.Helper()
	networkHash, err := hex.DecodeString(metadata.NetworkHash)
	if err != nil {
		t.Fatal(err)
	}
	copy(tip.Value.Proposal_header.Network_hash[:], networkHash)
	tip.Value.Proposal_header.Txs_hash = hashSingleHostedBootstrapTxSet(protoutil.HashSignedTx(&tx))
}

func TestHashSingleHostedBootstrapTxSetMatchesHogdMerkleTriePrefix(t *testing.T) {
	var txHash proto.TransactionHash
	for i := range txHash {
		txHash[i] = byte(i)
	}
	wantBytes, err := hex.DecodeString("e5bc3a60b899d43959e618e136d92812d7cca7313ea6bc79079fae3e1b5d9d60")
	if err != nil {
		t.Fatal(err)
	}
	var want proto.TxsHash
	copy(want[:], wantBytes)
	if got := hashSingleHostedBootstrapTxSet(txHash); got != want {
		t.Fatalf("transaction set hash = %x, want %x", got, want)
	}
}

func TestLoadHostedBootstrapMetadataRejectsWrongKind(t *testing.T) {
	path := writeHostedBootstrapMetadata(t, `{
  "schemaVersion": 1,
  "kind": "wrong",
  "networkHash": "`+strings.Repeat("00", 32)+`",
  "protocol": "`+testProtocolText()+`",
  "initializationTransactions": []
}`)

	if _, err := loadHostedBootstrapMetadata(path); err == nil {
		t.Fatal("expected wrong kind to fail")
	}
}

func TestLoadHostedBootstrapMetadataRejectsInvalidNetworkHash(t *testing.T) {
	for _, networkHash := range []string{
		"devnet",
		strings.Repeat("AA", 32),
		strings.Repeat("gg", 32),
	} {
		path := writeHostedBootstrapMetadata(t, `{
  "schemaVersion": 1,
  "kind": "hog-hosted-bootstrap",
  "networkHash": "`+networkHash+`"
}`)
		if _, err := loadHostedBootstrapMetadata(path); err == nil || !strings.Contains(err.Error(), "networkHash") {
			t.Fatalf("networkHash %q returned %v", networkHash, err)
		}
	}
}

func TestDecodeHostedBootstrapTransactionsDecodesXDR(t *testing.T) {
	tx := proto.SignedTransaction{}
	txHex := xdrHexSignedTransaction(t, tx)
	metadata := hostedBootstrapMetadata{
		SchemaVersion: 1,
		Kind:          "hog-hosted-bootstrap",
		NetworkHash:   strings.Repeat("00", 32),
		Protocol:      testProtocolText(),
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
		NetworkHash:   strings.Repeat("00", 32),
		Protocol:      testProtocolText(),
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
		NetworkHash:   strings.Repeat("00", 32),
		Protocol:      testProtocolText(),
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
		schedulerAddress:        "127.0.0.1:8070",
		schedulerGatewayAddress: "gateway.internal:8075",
		indexerAddress:          "127.0.0.1:8061",
		dbGatewayAddress:        "hog-db.internal:8084",
		viewGatewayAddress:      "hog-exec.internal:8042",
		publicRPCURL:            "http://203.0.113.10:8545",
		runtimeManifestOut:      path,
	}, hostedBootstrapCommit{Height: 1, ValueHash: "0x" + strings.Repeat("88", 32)})
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
	if manifest["schemaVersion"] != float64(2) {
		t.Fatalf("schemaVersion = %v", manifest["schemaVersion"])
	}
	if manifest["networkHash"] != strings.Repeat("00", 32) {
		t.Fatalf("networkHash = %v", manifest["networkHash"])
	}
	endpoints := manifest["endpoints"].(map[string]any)
	if endpoints["scheduler"] != "127.0.0.1:8070" ||
		endpoints["schedulerGateway"] != "gateway.internal:8075" ||
		endpoints["indexer"] != "127.0.0.1:8061" ||
		endpoints["dbGateway"] != "hog-db.internal:8084" ||
		endpoints["viewGateway"] != "hog-exec.internal:8042" {
		t.Fatalf("unexpected endpoints: %+v", endpoints)
	}
	hosted := manifest["hostedBootstrap"].(map[string]any)
	if hosted["publicRpcUrl"] != "http://203.0.113.10:8545" {
		t.Fatalf("unexpected hosted extension: %+v", hosted)
	}
}

func TestWriteHostedRuntimeManifestAllowsHostnameEndpoint(t *testing.T) {
	if _, err := writeHostedRuntimeManifest(testHostedBootstrapMetadata(), hostedBootstrapConfig{
		schedulerAddress:        "localhost:8070",
		schedulerGatewayAddress: "gateway.internal:8075",
		runtimeManifestOut:      filepath.Join(t.TempDir(), "runtime-manifest.json"),
	}, hostedBootstrapCommit{}); err != nil {
		t.Fatalf("expected hostname scheduler endpoint to pass: %v", err)
	}
}

func TestWriteHostedRuntimeManifestRejectsMalformedEndpoint(t *testing.T) {
	for _, tc := range []struct {
		address string
		wantErr string
	}{
		{address: "localhost:not-a-port", wantErr: "port must be numeric"},
		{address: "bad host:8070", wantErr: "host must be an IP literal or DNS hostname"},
		{address: "host_with_underscore:8070", wantErr: "host must be an IP literal or DNS hostname"},
	} {
		_, err := writeHostedRuntimeManifest(testHostedBootstrapMetadata(), hostedBootstrapConfig{
			schedulerAddress:        tc.address,
			schedulerGatewayAddress: "gateway.internal:8075",
			runtimeManifestOut:      filepath.Join(t.TempDir(), "runtime-manifest.json"),
		}, hostedBootstrapCommit{})
		if err == nil {
			t.Fatalf("expected %s to fail", tc.address)
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("unexpected error for %s: %v", tc.address, err)
		}
	}
}

func TestInitializeHostedBootstrapValidatesManifestConfigBeforeMetadata(t *testing.T) {
	err := initializeHostedBootstrap(context.Background(), hostedBootstrapConfig{
		metadataPath:            filepath.Join(t.TempDir(), "missing-metadata.json"),
		schedulerAddress:        "localhost:not-a-port",
		schedulerGatewayAddress: "gateway.internal:8075",
		runtimeManifestOut:      filepath.Join(t.TempDir(), "runtime-manifest.json"),
	})
	if err == nil {
		t.Fatal("expected malformed scheduler endpoint to fail")
	}
	if !strings.Contains(err.Error(), "port must be numeric") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommitHostedBootstrapRejectsMissingCeremonyTransaction(t *testing.T) {
	tip := proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipOK}
	*tip.Db_tip() = proto.DbTip{}
	ctl := &fakeHostedBootstrapCoordinator{tip: tip}
	err := commitHostedBootstrap(context.Background(), hostedBootstrapConfig{
		metadataPath:            writeHostedBootstrapMetadataJSON(t, testHostedBootstrapMetadata()),
		schedulerAddress:        "127.0.0.1:8070",
		schedulerGatewayAddress: "gateway.internal:8075",
	}, ctl)
	if err == nil || !strings.Contains(err.Error(), "exactly one initialization transaction") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitializeHostedBootstrapWithCoordinatorCommitsCeremonyBeforeSignal(t *testing.T) {
	tx := proto.SignedTransaction{}
	metadata := testHostedBootstrapMetadataWithTransactions(t, tx)
	metadataPath := writeHostedBootstrapMetadataJSON(t, metadata)
	manifestPath := filepath.Join(t.TempDir(), "runtime-manifest.json")
	tip := proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipOK}
	*tip.Db_tip() = proto.DbTip{}
	ctl := &fakeHostedBootstrapCoordinator{
		tip:    tip,
		result: testHostedBootstrapCeremonyResult(t, 1),
	}
	setHostedBootstrapCommitIdentity(t, metadata, tx, &ctl.result.Commit().Db_tip)

	if err := initializeHostedBootstrapWithCoordinator(context.Background(), hostedBootstrapConfig{
		metadataPath:            metadataPath,
		schedulerAddress:        "127.0.0.1:8070",
		schedulerGatewayAddress: "gateway.internal:8075",
		indexerAddress:          "127.0.0.1:8061",
		publicRPCURL:            "http://203.0.113.10:8545",
		runtimeManifestOut:      manifestPath,
	}, ctl); err != nil {
		t.Fatal(err)
	}
	if !ctl.signaled {
		t.Fatal("expected Baton genesis readiness to be signaled after the ceremony")
	}
	if ctl.request == nil {
		t.Fatal("expected genesis ceremony request")
	}
	if ctl.request.Expected_parent_height != 0 {
		t.Fatalf("expected parent height = %d, want 0", ctl.request.Expected_parent_height)
	}
	wantParentHash := protoutil.HashValueToProto(&tip.Db_tip().Value)
	if ctl.request.Expected_parent_value_hash != wantParentHash {
		t.Fatal("ceremony request parent hash does not match height-0 tip")
	}
	txs, err := xdrutil.SerializeSignedTransactionBatch(ctl.request.Transactions)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 1 || protoutil.HashSignedTx(&txs[0]) != protoutil.HashSignedTx(&tx) {
		t.Fatalf("ceremony request contained %d unexpected transactions", len(txs))
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"height": 1`)) {
		t.Fatalf("manifest does not include hosted bootstrap commit height: %s", data)
	}
}

func TestInitializeHostedBootstrapWithCoordinatorDoesNotSignalOnCeremonyFailure(t *testing.T) {
	tx := proto.SignedTransaction{}
	tip := proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipOK}
	*tip.Db_tip() = proto.DbTip{}
	ctl := &fakeHostedBootstrapCoordinator{
		tip:    tip,
		result: proto.GenesisCeremonyResult{Outcome: proto.GenesisCeremonyExecutionFailed},
	}
	err := initializeHostedBootstrapWithCoordinator(context.Background(), hostedBootstrapConfig{
		metadataPath:            writeHostedBootstrapMetadataJSON(t, testHostedBootstrapMetadataWithTransactions(t, tx)),
		schedulerAddress:        "127.0.0.1:8070",
		schedulerGatewayAddress: "gateway.internal:8075",
	}, ctl)
	if err == nil || !strings.Contains(err.Error(), "GenesisCeremonyExecutionFailed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctl.signaled {
		t.Fatal("Baton was signaled despite a failed genesis ceremony")
	}
}

func TestInitializeHostedBootstrapWithCoordinatorRecoversCommittedCeremony(t *testing.T) {
	tx := proto.SignedTransaction{}
	metadata := testHostedBootstrapMetadataWithTransactions(t, tx)
	tip := proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipOK}
	*tip.Db_tip() = proto.DbTip{}
	tip.Db_tip().Value.Proposal_header.Block_height = 1
	setHostedBootstrapCommitIdentity(t, metadata, tx, tip.Db_tip())
	ctl := &fakeHostedBootstrapCoordinator{tip: tip}

	err := initializeHostedBootstrapWithCoordinator(context.Background(), hostedBootstrapConfig{
		metadataPath:            writeHostedBootstrapMetadataJSON(t, metadata),
		schedulerAddress:        "127.0.0.1:8070",
		schedulerGatewayAddress: "gateway.internal:8075",
	}, ctl)
	if err != nil {
		t.Fatal(err)
	}
	if ctl.request != nil {
		t.Fatal("genesis ceremony was recommitted during height-1 recovery")
	}
	if !ctl.signaled {
		t.Fatal("expected Baton genesis readiness after height-1 recovery")
	}
}

func TestInitializeHostedBootstrapWithCoordinatorRejectsRecoveryFromDifferentNetwork(t *testing.T) {
	tx := proto.SignedTransaction{}
	metadata := testHostedBootstrapMetadataWithTransactions(t, tx)
	tip := proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipOK}
	*tip.Db_tip() = proto.DbTip{}
	tip.Db_tip().Value.Proposal_header.Block_height = 1
	setHostedBootstrapCommitIdentity(t, metadata, tx, tip.Db_tip())
	tip.Db_tip().Value.Proposal_header.Network_hash[0] ^= 0xff
	ctl := &fakeHostedBootstrapCoordinator{tip: tip}

	err := initializeHostedBootstrapWithCoordinator(context.Background(), hostedBootstrapConfig{
		metadataPath:            writeHostedBootstrapMetadataJSON(t, metadata),
		schedulerAddress:        "127.0.0.1:8070",
		schedulerGatewayAddress: "gateway.internal:8075",
	}, ctl)
	if err == nil || !strings.Contains(err.Error(), "network hash") {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctl.signaled {
		t.Fatal("Baton was signaled for an unrelated network")
	}
}

func TestInitializeHostedBootstrapWithCoordinatorRejectsRecoveryFromDifferentTransaction(t *testing.T) {
	committedTx := proto.SignedTransaction{}
	metadataTx := proto.SignedTransaction{}
	metadataTx.Tx.Uniquifier.Nonce = 1
	metadata := testHostedBootstrapMetadataWithTransactions(t, metadataTx)
	tip := proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipOK}
	*tip.Db_tip() = proto.DbTip{}
	tip.Db_tip().Value.Proposal_header.Block_height = 1
	setHostedBootstrapCommitIdentity(t, metadata, committedTx, tip.Db_tip())
	ctl := &fakeHostedBootstrapCoordinator{tip: tip}

	err := initializeHostedBootstrapWithCoordinator(context.Background(), hostedBootstrapConfig{
		metadataPath:            writeHostedBootstrapMetadataJSON(t, metadata),
		schedulerAddress:        "127.0.0.1:8070",
		schedulerGatewayAddress: "gateway.internal:8075",
	}, ctl)
	if err == nil || !strings.Contains(err.Error(), "transaction set hash") {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctl.signaled {
		t.Fatal("Baton was signaled for an unrelated genesis transaction")
	}
}

func TestHostedRuntimeManifestRequiresPublicSchedulerGatewayBeforeBootstrap(t *testing.T) {
	for _, address := range []string{"", "gateway.internal:not-a-port", "bad host:8075"} {
		t.Run(address, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "runtime-manifest.json")
			cfg := hostedBootstrapConfig{
				metadataPath:            filepath.Join(t.TempDir(), "missing-metadata.json"),
				schedulerAddress:        "coordinator.internal:8070",
				schedulerGatewayAddress: address,
				runtimeManifestOut:      path,
			}
			err := initializeHostedBootstrap(context.Background(), cfg)
			if err == nil || !strings.Contains(err.Error(), "schedulerGateway") {
				t.Fatalf("expected schedulerGateway validation before metadata or bootstrap, got %v", err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("invalid configuration wrote a runtime manifest: %v", err)
			}
		})
	}
}

func TestHostedBootstrapWithoutManifestDoesNotRequireSchedulerGateway(t *testing.T) {
	if err := validateHostedBootstrapManifestConfig(hostedBootstrapConfig{}); err != nil {
		t.Fatal(err)
	}
}
