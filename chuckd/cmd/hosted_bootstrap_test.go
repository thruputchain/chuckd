package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/thruputchain/chuckd/util/encoding"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/chuckd/util/xdrutil"
	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
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

// Match hogd's metadata producer directly, independently of the production
// hash helper, so a change to that helper cannot change both sides of a test.
func hostedBootstrapBodyHash(tx proto.SignedTransaction) proto.TransactionHash {
	var body bytes.Buffer
	libxdr.MarshalXdrWriter(&body, proto.XDR_Transaction(&tx.Tx))
	return proto.TransactionHash(sha256.Sum256(body.Bytes()))
}

func hostedBootstrapHashBase58(tx proto.SignedTransaction) string {
	hash := hostedBootstrapBodyHash(tx)
	return encoding.Base58(hash[:])
}

// testProtocolText is the protocol token as it appears in metadata JSON.
func testProtocolText() string {
	protocol := proto.HogProtocol()
	return string(protocol[:])
}

// The native nonce-zero account emitted for bootstrap seed "seed", index 0.
const testHostedBootstrapPublicKey = "9cnDtF1yBL5geTVVWx4Gp7y8aXe8M23QnT1uwSFepD1L"
const testHostedBootstrapWalletAddress = "3bhf6dpzbd31YRjkJNpfTaFekHmRpKNCoPxm699XgeCD"

func testHostedBootstrapMetadata() hostedBootstrapMetadata {
	return hostedBootstrapMetadata{
		SchemaVersion:              3,
		Kind:                       "hog-hosted-bootstrap",
		NetworkHash:                strings.Repeat("1", 32),
		Protocol:                   testProtocolText(),
		BootstrapAccountDerivation: json.RawMessage(`{"scheme":"groundhog-bootstrap-account-v1","seed":"seed"}`),
		BootstrapAccounts:          json.RawMessage(`[{"index":0,"role":"hosted_dev_account","publicKey":"` + testHostedBootstrapPublicKey + `","walletAddress":"` + testHostedBootstrapWalletAddress + `","initialStorageTokens":"1000"}]`),
		Bootstrap:                  json.RawMessage(`{"storageCostModel":{"hash":"93MB2qRDNVLxbmmPuYpLdAqn3u2x9ZhaVZK5wELHueP8","baseDataItemCost":"1","bytestringCostPerByte":"2","baseContractDeploymentCost":"3","contractDeploymentCostPerByte":"4","authPerKeyCost":"5","rcqCostPerEntry":"6","replayCacheCostPerEntry":"7","balanceCost":"8"}}`),
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
			Hash:  hostedBootstrapHashBase58(tx),
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
	networkHash, err := encoding.Decode32(metadata.NetworkHash)
	if err != nil {
		t.Fatal(err)
	}
	copy(tip.Value.Proposal_header.Network_hash[:], networkHash[:])
	tip.Value.Proposal_header.Txs_hash = hashSingleHostedBootstrapTxSet(hostedBootstrapBodyHash(tx))
}

func TestHashSingleHostedBootstrapTxSetMatchesHogdMerkleTriePrefix(t *testing.T) {
	var txHash proto.TransactionHash
	for i := range txHash {
		txHash[i] = byte(i)
	}
	wantBytes, err := hex.DecodeString("d88fb1d8dd2249974e30d358bd92e8e0aaff49abd476297573a791024759f726")
	if err != nil {
		t.Fatal(err)
	}
	var want proto.TxsHash
	copy(want[:], wantBytes)
	if got := hashSingleHostedBootstrapTxSet(txHash); got != want {
		t.Fatalf("transaction set hash = %x, want %x", got, want)
	}
}

func TestLoadHostedBootstrapMetadataRejectsLegacySchema(t *testing.T) {
	metadata := testHostedBootstrapMetadata()
	metadata.SchemaVersion = 2
	path := writeHostedBootstrapMetadataJSON(t, metadata)
	if _, err := loadHostedBootstrapMetadata(path); err == nil || !strings.Contains(err.Error(), "schemaVersion 2") {
		t.Fatalf("legacy metadata must require regeneration, got %v", err)
	}
}

func TestLoadHostedBootstrapMetadataRejectsWrongKind(t *testing.T) {
	path := writeHostedBootstrapMetadata(t, `{
  "schemaVersion": 3,
  "kind": "wrong",
  "networkHash": "`+strings.Repeat("1", 32)+`",
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
  "schemaVersion": 3,
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
		SchemaVersion: 3,
		Kind:          "hog-hosted-bootstrap",
		NetworkHash:   strings.Repeat("1", 32),
		Protocol:      testProtocolText(),
		InitializationTransactions: []struct {
			Index int    `json:"index"`
			Hash  string `json:"hash"`
			XDR   string `json:"xdr"`
		}{
			{Index: 0, Hash: hostedBootstrapHashBase58(tx), XDR: txHex},
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
		SchemaVersion: 3,
		Kind:          "hog-hosted-bootstrap",
		NetworkHash:   strings.Repeat("1", 32),
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
		SchemaVersion: 3,
		Kind:          "hog-hosted-bootstrap",
		NetworkHash:   strings.Repeat("1", 32),
		Protocol:      testProtocolText(),
		InitializationTransactions: []struct {
			Index int    `json:"index"`
			Hash  string `json:"hash"`
			XDR   string `json:"xdr"`
		}{
			{Index: 0, Hash: strings.Repeat("1", 32), XDR: txHex},
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
		schedulerGatewayAddress: "127.0.0.1:8075",
		indexerAddress:          "127.0.0.1:8061",
		dbGatewayAddress:        "hog-db.internal:8084",
		viewGatewayAddress:      "hog-exec.internal:8042",
		publicRPCURL:            "http://203.0.113.10:8545",
		runtimeManifestOut:      path,
	}, hostedBootstrapCommit{Height: 1, ValueHash: "AByCTxLPRZPoyK22KdMxa3xkCbcNbeNWzVeEvh6UcJs9"})
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
	if manifest["schemaVersion"] != float64(4) {
		t.Fatalf("schemaVersion = %v", manifest["schemaVersion"])
	}
	if manifest["networkHash"] != strings.Repeat("1", 32) {
		t.Fatalf("networkHash = %v", manifest["networkHash"])
	}
	wantDerivation := map[string]any{
		"scheme": "groundhog-bootstrap-account-v1",
		"seed":   "seed",
	}
	if !reflect.DeepEqual(manifest["bootstrapAccountDerivation"], wantDerivation) {
		t.Fatalf("unexpected account derivation: %+v", manifest["bootstrapAccountDerivation"])
	}
	wantAccounts := []any{map[string]any{
		"index":                float64(0),
		"role":                 "hosted_dev_account",
		"publicKey":            testHostedBootstrapPublicKey,
		"walletAddress":        testHostedBootstrapWalletAddress,
		"initialStorageTokens": "1000",
	}}
	if !reflect.DeepEqual(manifest["bootstrapAccounts"], wantAccounts) {
		t.Fatalf("unexpected native bootstrap accounts: %+v", manifest["bootstrapAccounts"])
	}
	bootstrap := manifest["bootstrap"].(map[string]any)
	if len(bootstrap) != 1 || bootstrap["storageCostModel"] == nil {
		t.Fatalf("unexpected bootstrap metadata: %+v", bootstrap)
	}
	endpoints := manifest["endpoints"].(map[string]any)
	if endpoints["scheduler"] != "127.0.0.1:8070" ||
		endpoints["schedulerGateway"] != "127.0.0.1:8075" ||
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
		schedulerGatewayAddress: "127.0.0.1:8075",
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
			schedulerGatewayAddress: "127.0.0.1:8075",
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
		schedulerGatewayAddress: "127.0.0.1:8075",
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
		schedulerGatewayAddress: "127.0.0.1:8075",
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
		schedulerGatewayAddress: "127.0.0.1:8075",
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
		schedulerGatewayAddress: "127.0.0.1:8075",
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
		schedulerGatewayAddress: "127.0.0.1:8075",
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
		schedulerGatewayAddress: "127.0.0.1:8075",
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
		schedulerGatewayAddress: "127.0.0.1:8075",
	}, ctl)
	if err == nil || !strings.Contains(err.Error(), "transaction set hash") {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctl.signaled {
		t.Fatal("Baton was signaled for an unrelated genesis transaction")
	}
}

func TestHostedRuntimeManifestRequiresPublicSchedulerGatewayBeforeBootstrap(t *testing.T) {
	for _, address := range []string{"", "127.0.0.1:not-a-port", "bad host:8075", "gateway.internal:8075"} {
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

func TestHostedRuntimeManifestAcceptsIPv6SchedulerGateway(t *testing.T) {
	if err := validateHostedBootstrapManifestConfig(hostedBootstrapConfig{
		schedulerAddress:        "127.0.0.1:8070",
		schedulerGatewayAddress: "[::1]:8075",
		runtimeManifestOut:      "runtime-manifest.json",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeHostedBootstrapTransactionsRejectsSignedEnvelopeHash(t *testing.T) {
	tx := proto.SignedTransaction{Tx: proto.Transaction{Uniquifier: proto.Uniquifier{Nonce: 7}}}
	metadata := testHostedBootstrapMetadataWithTransactions(t, tx)
	signedHash := protoutil.HashSignedTx(&tx)
	metadata.InitializationTransactions[0].Hash = encoding.Base58(signedHash[:])
	if _, err := decodeHostedBootstrapTransactions(metadata); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("signed-envelope metadata hash: got %v, want hash mismatch", err)
	}
}

func TestValidateHostedBootstrapCommittedTipRequiresBodyHashRoot(t *testing.T) {
	tx := proto.SignedTransaction{Tx: proto.Transaction{Uniquifier: proto.Uniquifier{Nonce: 7}}}
	metadata := testHostedBootstrapMetadataWithTransactions(t, tx)
	var tip proto.DbTip
	setHostedBootstrapCommitIdentity(t, metadata, tx, &tip)
	if err := validateHostedBootstrapCommittedTip(metadata, tx, &tip); err != nil {
		t.Fatalf("body-hash root: %v", err)
	}
	tip.Value.Proposal_header.Txs_hash = hashSingleHostedBootstrapTxSet(proto.TransactionHash(protoutil.HashSignedTx(&tx)))
	if err := validateHostedBootstrapCommittedTip(metadata, tx, &tip); err == nil || !strings.Contains(err.Error(), "transaction set hash") {
		t.Fatalf("signed-envelope root: got %v, want transaction set hash mismatch", err)
	}
}
