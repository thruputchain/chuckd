package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/groundhogchain/chuckd/chuckd/cmdkit"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/spf13/cobra"
)

var (
	hostedBootstrapCmd = &cobra.Command{
		Use:   "hosted-bootstrap",
		Short: "hosted devnet bootstrap helpers",
	}

	hostedBootstrapInitializeCmd = &cobra.Command{
		Use:   "initialize",
		Short: "submit and commit hosted devnet initialization transactions",
		Long:  "Submit and commit hosted devnet initialization transactions during the fenced bootstrap phase. Run this after genesis is loaded and before normal coordinators start.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdkit.Invoke(ctx, func() {
				must.NoError(ctx, initializeHostedBootstrap(ctx, hostedBootstrapMetadataPath, hostedBootstrapSchedulerAddress))
			})
		},
	}
)

var (
	hostedBootstrapMetadataPath     string
	hostedBootstrapSchedulerAddress string
)

type hostedBootstrapMetadata struct {
	SchemaVersion              int    `json:"schemaVersion"`
	Kind                       string `json:"kind"`
	NetworkID                  string `json:"networkId"`
	Protocol                   string `json:"protocol"`
	GenesisHash                string `json:"genesisHash"`
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
	if metadata.SchemaVersion != 1 {
		return hostedBootstrapMetadata{}, fmt.Errorf("unsupported hosted bootstrap metadata schemaVersion %d", metadata.SchemaVersion)
	}
	if metadata.Kind != "hog-hosted-bootstrap" {
		return hostedBootstrapMetadata{}, fmt.Errorf("unsupported hosted bootstrap metadata kind %q", metadata.Kind)
	}
	if metadata.NetworkID == "" {
		return hostedBootstrapMetadata{}, fmt.Errorf("hosted bootstrap metadata missing networkId")
	}
	if metadata.Protocol != proto.HogProtocol() {
		return hostedBootstrapMetadata{}, fmt.Errorf("hosted bootstrap metadata protocol %q does not match chuckd protocol %q", metadata.Protocol, proto.HogProtocol())
	}
	if metadata.GenesisHash == "" {
		return hostedBootstrapMetadata{}, fmt.Errorf("hosted bootstrap metadata missing genesisHash")
	}
	return metadata, nil
}

func decodeHostedBootstrapHash(value string, index int) (proto.SignedTransactionHash, error) {
	trimmed := strings.TrimPrefix(value, "0x")
	if len(trimmed) != 64 {
		return proto.SignedTransactionHash{}, fmt.Errorf("initialization transaction %d hash must be 32 bytes of hex", index)
	}
	decoded, err := hex.DecodeString(trimmed)
	if err != nil {
		return proto.SignedTransactionHash{}, fmt.Errorf("decoding initialization transaction %d hash: %w", index, err)
	}
	var hash proto.SignedTransactionHash
	copy(hash[:], decoded)
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
		if actualHash := protoutil.HashSignedTx(&tx); actualHash != expectedHash {
			return nil, fmt.Errorf("initialization transaction %d hash mismatch", i)
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func initializeHostedBootstrap(ctx context.Context, metadataPath string, schedulerAddress string) error {
	metadata, err := loadHostedBootstrapMetadata(metadataPath)
	if err != nil {
		return err
	}
	txs, err := decodeHostedBootstrapTransactions(metadata)
	if err != nil {
		return err
	}

	client, err := sched.DialSchedulerCoordClient(ctx, libnet.TCPNet{}, schedulerAddress)
	if err != nil {
		return fmt.Errorf("connecting to scheduler at %s: %w", schedulerAddress, err)
	}
	defer client.Close()

	tip, err := client.Schedcoord_get_tip(ctx)
	if err != nil {
		return fmt.Errorf("getting scheduler tip: %w", err)
	}
	if tip.Outcome != proto.SchedulerGetTipOK {
		return fmt.Errorf("scheduler has no loaded genesis for hosted bootstrap: %v", tip.Outcome)
	}
	if tip.Db_tip().Height > 0 {
		return fmt.Errorf("hosted bootstrap must run at genesis height 0; scheduler is already at height %d", tip.Db_tip().Height)
	}
	if len(txs) == 0 {
		liblog.Infof(ctx, "hosted bootstrap has no initialization transactions network=%s genesis=%s", metadata.NetworkID, metadata.GenesisHash)
		return nil
	}

	batch := xdrutil.ParallelizeSignedTransactionBatch(
		libxdr.NewXdrMarshaler(), systemfixed.UnvettedTxBatchMaxSegments, txs)
	if err := client.Schedulercoord_submit_unvetted(ctx, batch); err != nil {
		return fmt.Errorf("submitting hosted bootstrap initialization transactions: %w", err)
	}

	nomination, err := client.Schedulercoord_nominate(ctx, *tip.Db_tip())
	if err != nil {
		return fmt.Errorf("nominating hosted bootstrap initialization block: %w", err)
	}
	if nomination.Outcome != proto.SchedulerNominationOK {
		return fmt.Errorf("hosted bootstrap nomination failed: %v", nomination.Outcome)
	}
	commitOutcome, err := client.Schedulercoord_commit_block(ctx, nomination.Nominated_block().Block_id)
	if err != nil {
		return fmt.Errorf("committing hosted bootstrap initialization block: %w", err)
	}
	if commitOutcome != proto.SchedulerCommitOK {
		return fmt.Errorf("hosted bootstrap commit failed: %v", commitOutcome)
	}

	liblog.Infof(ctx, "hosted bootstrap initialized network=%s genesis=%s txs=%d height=%d block_hash=%s",
		metadata.NetworkID, metadata.GenesisHash, len(txs), nomination.Nominated_block().Nomination.Block_height,
		libxdr.HashString(nomination.Nominated_block().Nomination.Block_hash))
	return nil
}

func init() {
	sysCmd.AddCommand(hostedBootstrapCmd)
	hostedBootstrapCmd.AddCommand(hostedBootstrapInitializeCmd)
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapMetadataPath, "metadata", "", "hosted bootstrap metadata generated by hog-hosted-bootstrap")
	hostedBootstrapInitializeCmd.Flags().StringVar(&hostedBootstrapSchedulerAddress, "scheduler", "127.0.0.1:8070", "scheduler coordinator address")
}
