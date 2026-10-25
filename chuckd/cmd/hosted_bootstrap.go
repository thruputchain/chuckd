package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thruputchain/chuckd"
	"github.com/thruputchain/chuckd/chuckd/cmdkit"
	"github.com/thruputchain/chuckd/sub/sched"
	"github.com/thruputchain/chuckd/util/encoding"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/chuckd/util/xdrutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

var (
	hostedBootstrapCmd = &cobra.Command{
		Use:   "hosted-bootstrap",
		Short: "hosted devnet bootstrap helpers",
	}

	hostedBootstrapInitializeCmd = &cobra.Command{
		Use:   "initialize",
		Short: "commit the hosted devnet genesis ceremony",
		Long:  "Commit the hosted devnet initialization transaction atomically through the scheduler's genesis ceremony RPC.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdkit.Invoke(ctx, func() {
				must.NoError(ctx, initializeHostedBootstrap(ctx, hostedBootstrapConfig{
					metadataPath:            hostedBootstrapMetadataPath,
					schedulerAddress:        hostedBootstrapSchedulerAddress,
					schedulerGatewayAddress: hostedBootstrapSchedulerGatewayAddress,
					indexerAddress:          hostedBootstrapIndexerAddress,
					dbGatewayAddress:        hostedBootstrapDbGatewayAddress,
					viewGatewayAddress:      hostedBootstrapViewGatewayAddress,
					publicRPCURL:            hostedBootstrapPublicRPCURL,
					runtimeManifestOut:      hostedBootstrapRuntimeManifestOut,
				}))
			})
		},
	}
)

var (
	hostedBootstrapMetadataPath            string
	hostedBootstrapSchedulerAddress        string
	hostedBootstrapSchedulerGatewayAddress string
	hostedBootstrapIndexerAddress          string
	hostedBootstrapDbGatewayAddress        string
	hostedBootstrapViewGatewayAddress      string
	hostedBootstrapPublicRPCURL            string
	hostedBootstrapRuntimeManifestOut      string
)

type hostedBootstrapConfig struct {
	metadataPath            string
	schedulerAddress        string
	schedulerGatewayAddress string
	indexerAddress          string
	dbGatewayAddress        string
	viewGatewayAddress      string
	publicRPCURL            string
	runtimeManifestOut      string
}

type hostedBootstrapCoordinator interface {
	GetTip(context.Context) (proto.SchedulerGetTipResult, error)
	CommitGenesisCeremony(context.Context, proto.GenesisCeremonyRequest) (proto.GenesisCeremonyResult, error)
	SignalGenesisIsReady()
}

type hostedBootstrapScheduler interface {
	GetTip(context.Context) (proto.SchedulerGetTipResult, error)
	CommitGenesisCeremony(context.Context, proto.GenesisCeremonyRequest) (proto.GenesisCeremonyResult, error)
}

type directHostedBootstrapScheduler struct {
	conn sched.SchedulerCoordConn
}

func (x directHostedBootstrapScheduler) GetTip(ctx context.Context) (proto.SchedulerGetTipResult, error) {
	return x.conn.Schedcoord_get_tip(ctx)
}

func (x directHostedBootstrapScheduler) CommitGenesisCeremony(ctx context.Context, request proto.GenesisCeremonyRequest) (proto.GenesisCeremonyResult, error) {
	return x.conn.Schedulercoord_commit_genesis_ceremony(ctx, request)
}

type hostedBootstrapMetadata struct {
	SchemaVersion              int             `json:"schemaVersion"`
	Kind                       string          `json:"kind"`
	NetworkHash                string          `json:"networkHash"`
	Protocol                   string          `json:"protocol"`
	BootstrapAccountDerivation json.RawMessage `json:"bootstrapAccountDerivation"`
	BootstrapAccounts          json.RawMessage `json:"bootstrapAccounts"`
	Bootstrap                  json.RawMessage `json:"bootstrap"`
	InitializationTransactions []struct {
		Index int    `json:"index"`
		Hash  string `json:"hash"`
		XDR   string `json:"xdr"`
	} `json:"initializationTransactions"`
}

func loadHostedBootstrapMetadata(path string) (hostedBootstrapMetadata, error) {
	if path == "" {
		return hostedBootstrapMetadata{}, fmt.Errorf("--metadata is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return hostedBootstrapMetadata{}, fmt.Errorf("reading hosted bootstrap metadata: %w", err)
	}
	var metadata hostedBootstrapMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return hostedBootstrapMetadata{}, fmt.Errorf("parsing hosted bootstrap metadata: %w", err)
	}
	if metadata.SchemaVersion != 3 {
		return hostedBootstrapMetadata{}, fmt.Errorf("unsupported hosted bootstrap metadata schemaVersion %d", metadata.SchemaVersion)
	}
	if metadata.Kind != "hog-hosted-bootstrap" {
		return hostedBootstrapMetadata{}, fmt.Errorf("unsupported hosted bootstrap metadata kind %q", metadata.Kind)
	}
	if _, err := encoding.Decode32(metadata.NetworkHash); err != nil {
		return hostedBootstrapMetadata{}, fmt.Errorf("hosted bootstrap metadata networkHash: %w", err)
	}
	protocol := proto.HogProtocol()
	if metadata.Protocol != string(protocol[:]) {
		return hostedBootstrapMetadata{}, fmt.Errorf("hosted bootstrap metadata protocol %q does not match chuckd protocol %q", metadata.Protocol, string(protocol[:]))
	}
	if len(metadata.BootstrapAccountDerivation) == 0 {
		return hostedBootstrapMetadata{}, fmt.Errorf("hosted bootstrap metadata missing bootstrapAccountDerivation")
	}
	if len(metadata.BootstrapAccounts) == 0 {
		return hostedBootstrapMetadata{}, fmt.Errorf("hosted bootstrap metadata missing bootstrapAccounts")
	}
	if len(metadata.Bootstrap) == 0 {
		return hostedBootstrapMetadata{}, fmt.Errorf("hosted bootstrap metadata missing bootstrap")
	}
	return metadata, nil
}

func decodeHostedBootstrapHash(value string, index int) (proto.TransactionHash, error) {
	decoded, err := encoding.Decode32(value)
	if err != nil {
		return proto.TransactionHash{}, fmt.Errorf("decoding initialization transaction %d hash: %w", index, err)
	}
	var hash proto.TransactionHash
	copy(hash[:], decoded[:])
	return hash, nil
}

func decodeHostedBootstrapTransactions(metadata hostedBootstrapMetadata) ([]proto.SignedTransaction, error) {
	txs := make([]proto.SignedTransaction, 0, len(metadata.InitializationTransactions))
	for i, entry := range metadata.InitializationTransactions {
		if entry.Index != i {
			return nil, fmt.Errorf("initialization transaction index %d is out of order at position %d", entry.Index, i)
		}
		if entry.Hash == "" {
			return nil, fmt.Errorf("initialization transaction %d missing hash", i)
		}
		if entry.XDR == "" {
			return nil, fmt.Errorf("initialization transaction %d missing xdr", i)
		}
		expectedHash, err := decodeHostedBootstrapHash(entry.Hash, i)
		if err != nil {
			return nil, err
		}
		txBytes, err := hex.DecodeString(entry.XDR)
		if err != nil {
			return nil, fmt.Errorf("decoding initialization transaction %d xdr: %w", i, err)
		}
		var tx proto.SignedTransaction
		if err := libxdr.UnmarshalXdrEOF(&tx, txBytes); err != nil {
			return nil, fmt.Errorf("unmarshalling initialization transaction %d xdr: %w", i, err)
		}
		if actualHash := protoutil.HashTxBody(&tx.Tx); actualHash != expectedHash {
			return nil, fmt.Errorf("initialization transaction %d hash mismatch", i)
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

type hostedBootstrapCommit struct {
	Height    proto.BlockHeight `json:"height"`
	ValueHash string            `json:"valueHash"`
}

// hashSingleHostedBootstrapTxSet reproduces the MerkleTriePrefix root used by
// hogd for a transaction set containing one key with an empty value. Hosted
// bootstrap deliberately requires exactly one ceremony transaction, so this
// small, fixed-shape implementation avoids introducing a second general
// Merkle-trie implementation in Go.
func hashSingleHostedBootstrapTxSet(txHash proto.TransactionHash) proto.TxsHash {
	const (
		branchNodeType = byte(0)
		leafNodeType   = byte(2)
		numBranches    = 16
	)

	hashBranch := func(children [numBranches][sha256.Size]byte) [sha256.Size]byte {
		// A branch encoding is its type, a false value-present byte, and its
		// sixteen child hashes.
		encoding := make([]byte, 0, 2+numBranches*sha256.Size)
		encoding = append(encoding, branchNodeType, 0)
		for _, child := range children {
			encoding = append(encoding, child[:]...)
		}
		return sha256.Sum256(encoding)
	}

	// Leaves retain the full key, including both routing nibbles consumed by
	// MerkleTriePrefix and the selected inner trie's root branch.
	leafEncoding := make([]byte, 0, 1+8+sha256.Size+1+sha256.Size)
	leafEncoding = append(leafEncoding, leafNodeType)
	pathLength := make([]byte, 8)
	binary.LittleEndian.PutUint64(pathLength, 64)
	leafEncoding = append(leafEncoding, pathLength...)
	leafEncoding = append(leafEncoding, txHash[:]...)
	leafEncoding = append(leafEncoding, 1)
	emptyValueHash := sha256.Sum256(nil)
	leafEncoding = append(leafEncoding, emptyValueHash[:]...)
	leafHash := sha256.Sum256(leafEncoding)

	var innerChildren [numBranches][sha256.Size]byte
	innerChildren[txHash[0]&0x0f] = leafHash
	innerRootHash := hashBranch(innerChildren)

	emptyInnerRootHash := hashBranch([numBranches][sha256.Size]byte{})
	var outerChildren [numBranches][sha256.Size]byte
	for i := range outerChildren {
		outerChildren[i] = emptyInnerRootHash
	}
	outerChildren[txHash[0]>>4] = innerRootHash
	return proto.TxsHash(hashBranch(outerChildren))
}

func validateHostedBootstrapCommittedTip(metadata hostedBootstrapMetadata, tx proto.SignedTransaction, tip *proto.DbTip) error {
	networkHashBytes, err := encoding.Decode32(metadata.NetworkHash)
	if err != nil {
		return fmt.Errorf("decoding hosted bootstrap network hash: %w", err)
	}
	var expectedNetworkHash proto.NetworkHash
	copy(expectedNetworkHash[:], networkHashBytes[:])
	if actual := tip.Value.Proposal_header.Network_hash; actual != expectedNetworkHash {
		return fmt.Errorf("committed genesis ceremony network hash %s does not match hosted bootstrap metadata %s", encoding.Base58(actual[:]), encoding.Base58(expectedNetworkHash[:]))
	}
	expectedTxsHash := hashSingleHostedBootstrapTxSet(protoutil.HashTxBody(&tx.Tx))
	if actual := tip.Value.Proposal_header.Txs_hash; actual != expectedTxsHash {
		return fmt.Errorf("committed genesis ceremony transaction set hash %s does not match hosted bootstrap transaction %s", encoding.Base58(actual[:]), encoding.Base58(expectedTxsHash[:]))
	}
	return nil
}

func runtimeManifestProducer() map[string]any {
	producer := map[string]any{
		"binary":  "chuckd",
		"version": chuckd.Short(),
	}
	if revision := chuckd.GetVersionInfo().Revision; revision != "" {
		producer["commitHash"] = revision
	}
	return producer
}

func rawJSONField(name string, raw json.RawMessage) (any, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("parsing hosted bootstrap metadata %s: %w", name, err)
	}
	return value, nil
}

func validateSocketAddress(name string, address string) error {
	if address == "" {
		return nil
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid %s address %q: %w", name, address, err)
	}
	if host == "" {
		return fmt.Errorf("invalid %s address %q: host is required", name, address)
	}
	if !isValidSocketHost(host) {
		return fmt.Errorf("invalid %s address %q: host must be an IP literal or DNS hostname", name, address)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return fmt.Errorf("invalid %s address %q: port must be numeric", name, address)
	}
	return nil
}

func isValidSocketHost(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	if host == "" || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func validateHostedBootstrapManifestConfig(cfg hostedBootstrapConfig) error {
	if cfg.runtimeManifestOut == "" {
		return nil
	}
	if err := validateSocketAddress("scheduler", cfg.schedulerAddress); err != nil {
		return err
	}
	if cfg.schedulerGatewayAddress == "" {
		return fmt.Errorf("schedulerGateway address is required when writing a runtime manifest")
	}
	if err := validateSocketAddress("schedulerGateway", cfg.schedulerGatewayAddress); err != nil {
		return err
	}
	host, _, _ := net.SplitHostPort(cfg.schedulerGatewayAddress)
	if net.ParseIP(host) == nil {
		return fmt.Errorf("invalid schedulerGateway address %q: host must be an IP literal", cfg.schedulerGatewayAddress)
	}
	if err := validateSocketAddress("indexer", cfg.indexerAddress); err != nil {
		return err
	}
	if cfg.dbGatewayAddress != "" {
		if err := validateSocketAddress("dbGateway", cfg.dbGatewayAddress); err != nil {
			return err
		}
	}
	if cfg.viewGatewayAddress != "" {
		if err := validateSocketAddress("viewGateway", cfg.viewGatewayAddress); err != nil {
			return err
		}
	}
	return nil
}

func writeJSONFileAtomically(path string, value any) ([32]byte, error) {
	if path == "" {
		return [32]byte{}, nil
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return [32]byte{}, err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return [32]byte{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return [32]byte{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return [32]byte{}, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return [32]byte{}, err
	}
	if err := tmp.Close(); err != nil {
		return [32]byte{}, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(data), nil
}

func writeHostedRuntimeManifest(metadata hostedBootstrapMetadata, cfg hostedBootstrapConfig, commit hostedBootstrapCommit) ([32]byte, error) {
	if cfg.runtimeManifestOut == "" {
		return [32]byte{}, nil
	}
	if err := validateHostedBootstrapManifestConfig(cfg); err != nil {
		return [32]byte{}, err
	}
	derivation, err := rawJSONField("bootstrapAccountDerivation", metadata.BootstrapAccountDerivation)
	if err != nil {
		return [32]byte{}, err
	}
	accounts, err := rawJSONField("bootstrapAccounts", metadata.BootstrapAccounts)
	if err != nil {
		return [32]byte{}, err
	}
	bootstrap, err := rawJSONField("bootstrap", metadata.Bootstrap)
	if err != nil {
		return [32]byte{}, err
	}
	endpoints := map[string]any{"scheduler": cfg.schedulerAddress, "schedulerGateway": cfg.schedulerGatewayAddress}
	if cfg.indexerAddress != "" {
		endpoints["indexer"] = cfg.indexerAddress
	}
	if cfg.dbGatewayAddress != "" {
		endpoints["dbGateway"] = cfg.dbGatewayAddress
	}
	if cfg.viewGatewayAddress != "" {
		endpoints["viewGateway"] = cfg.viewGatewayAddress
	}
	manifest := map[string]any{
		"schemaVersion":              4,
		"kind":                       "hosted-devnet-runtime",
		"networkHash":                metadata.NetworkHash,
		"protocol":                   metadata.Protocol,
		"producer":                   runtimeManifestProducer(),
		"endpoints":                  endpoints,
		"bootstrapAccountDerivation": derivation,
		"bootstrapAccounts":          accounts,
		"bootstrap":                  bootstrap,
		"hostedBootstrap": map[string]any{
			"publicRpcUrl": cfg.publicRPCURL,
			"commit":       commit,
		},
	}
	return writeJSONFileAtomically(cfg.runtimeManifestOut, manifest)
}

func initializeHostedBootstrap(ctx context.Context, cfg hostedBootstrapConfig) error {
	if err := validateHostedBootstrapManifestConfig(cfg); err != nil {
		return err
	}

	client, err := sched.DialSchedulerCoordClient(ctx, libnet.TCPNet{}, cfg.schedulerAddress)
	if err != nil {
		return fmt.Errorf("connecting to scheduler at %s: %w", cfg.schedulerAddress, err)
	}
	defer client.Close()
	return commitHostedBootstrap(ctx, cfg, directHostedBootstrapScheduler{conn: client})
}

func commitHostedBootstrap(ctx context.Context, cfg hostedBootstrapConfig, scheduler hostedBootstrapScheduler) error {
	if err := validateHostedBootstrapManifestConfig(cfg); err != nil {
		return err
	}
	metadata, err := loadHostedBootstrapMetadata(cfg.metadataPath)
	if err != nil {
		return err
	}
	txs, err := decodeHostedBootstrapTransactions(metadata)
	if err != nil {
		return err
	}

	tip, err := scheduler.GetTip(ctx)
	if err != nil {
		return fmt.Errorf("getting scheduler tip: %w", err)
	}
	if tip.Outcome != proto.SchedulerGetTipOK {
		return fmt.Errorf("scheduler has no loaded genesis for hosted bootstrap: %v", tip.Outcome)
	}
	if len(txs) != 1 {
		return fmt.Errorf("hosted bootstrap requires exactly one initialization transaction, got %d", len(txs))
	}

	committedTip := *tip.Db_tip()
	switch height := protoutil.ValueHeight(&committedTip.Value); height {
	case 0:
		parentHash := protoutil.HashValueToProto(&committedTip.Value)
		parallel := xdrutil.ParallelizeSignedTransactionBatch(libxdr.NewXdrMarshaler(), 1, txs)
		result, err := scheduler.CommitGenesisCeremony(ctx, proto.GenesisCeremonyRequest{
			Expected_parent_height:     0,
			Expected_parent_value_hash: parentHash,
			Context:                    proto.ProposalContext{Protocol: 0},
			Transactions:               parallel,
		})
		if err != nil {
			return fmt.Errorf("committing hosted bootstrap genesis ceremony: %w", err)
		}
		if result.Outcome != proto.GenesisCeremonyCommitted {
			return fmt.Errorf("hosted bootstrap genesis ceremony failed with %v", result.Outcome)
		}
		committedTip = result.Commit().Db_tip
		if committedHeight := protoutil.ValueHeight(&committedTip.Value); committedHeight != 1 {
			return fmt.Errorf("hosted bootstrap genesis ceremony committed height %d, want 1", committedHeight)
		}
		if err := validateHostedBootstrapCommittedTip(metadata, txs[0], &committedTip); err != nil {
			return err
		}
	case 1:
		if err := validateHostedBootstrapCommittedTip(metadata, txs[0], &committedTip); err != nil {
			return fmt.Errorf("recovering hosted bootstrap genesis ceremony: %w", err)
		}
		liblog.Infof(ctx, "hosted bootstrap genesis ceremony is already committed")
	default:
		return fmt.Errorf("hosted bootstrap must run at height 0 or recover height 1; scheduler is already at height %d", height)
	}

	valueHash := protoutil.HashValueToProto(&committedTip.Value)
	commit := hostedBootstrapCommit{Height: 1, ValueHash: encoding.Base58(valueHash[:])}

	manifestHash, err := writeHostedRuntimeManifest(metadata, cfg, commit)
	if err != nil {
		return fmt.Errorf("writing hosted runtime manifest: %w", err)
	}
	liblog.Infof(ctx, "hosted bootstrap initialized network=%s txs=%d height=%d value_hash=%s",
		metadata.NetworkHash, len(txs), commit.Height, commit.ValueHash)
	if cfg.runtimeManifestOut != "" {
		liblog.Infof(ctx, "hosted runtime manifest written path=%s hash=%s", cfg.runtimeManifestOut, "0x"+hex.EncodeToString(manifestHash[:]))
	}
	return nil
}

func initializeHostedBootstrapWithCoordinator(ctx context.Context, cfg hostedBootstrapConfig, ctl hostedBootstrapCoordinator) error {
	if err := commitHostedBootstrap(ctx, cfg, ctl); err != nil {
		return err
	}
	ctl.SignalGenesisIsReady()
	return nil
}

func init() {
	sysCmd.AddCommand(hostedBootstrapCmd)
	hostedBootstrapCmd.AddCommand(hostedBootstrapInitializeCmd)
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapMetadataPath, "metadata", "", "hosted bootstrap metadata generated by hog-hosted-bootstrap")
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapSchedulerAddress, "scheduler", "127.0.0.1:8070", "scheduler coordinator address")
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapSchedulerGatewayAddress, "scheduler-gateway", "", "public scheduler gateway IP:port; required with --runtime-manifest-out")
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapIndexerAddress, "indexer", "127.0.0.1:8061", "indexer address for the hosted runtime manifest")
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapDbGatewayAddress, "db-gateway", "", "DbGateway address for the hosted runtime manifest")
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapViewGatewayAddress, "view-gateway", "", "ViewGateway address for the hosted runtime manifest")
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapPublicRPCURL, "public-rpc-url", "", "public RPC URL to include in the hosted runtime manifest")
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapRuntimeManifestOut, "runtime-manifest-out", "", "write hosted runtime manifest after bootstrap commit")
}
