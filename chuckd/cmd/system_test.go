package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thruputchain/chuckd/util/encoding"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

var expectedMinimalCostModel = proto.StorageCostModelXdr{
	Base_data_item_cost:               12000,
	Bytestring_cost_per_byte:          10,
	Base_contract_deployment_cost:     12000,
	Contract_deployment_cost_per_byte: 10,
	Auth_per_key_cost:                 1111,
	Rcq_cost_per_entry:                8,
	Replay_cache_cost_per_entry:       7,
	Balance_cost:                      1234,
}

func writeMinimalGenesisFile(t *testing.T) string {
	t.Helper()
	genesis := proto.GenesisConfiguration{
		Network_hash:       proto.NetworkHash{0xab},
		Storage_cost_model: expectedMinimalCostModel,
		Root_contract:      proto.Contract{Contract_type: proto.KRv32},
	}
	genesis.Root_contract.Rv32_contract().Data = []byte{1, 2, 3}
	data, err := libxdr.XdrToJson(&genesis)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "genesis.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveNetworkHash(t *testing.T) {
	genesisPath := writeMinimalGenesisFile(t)
	metadata := testHostedBootstrapMetadata()
	var metadataHash proto.NetworkHash
	for i := range metadataHash {
		metadataHash[i] = 0xcd
	}
	metadata.NetworkHash = encoding.Base58(metadataHash[:])
	metadataPath := writeHostedBootstrapMetadataJSON(t, metadata)
	genesisHash := proto.NetworkHash{0xab}
	for _, tc := range []struct {
		name, genesis, metadata string
		want                    *proto.NetworkHash
		wantErr                 bool
	}{
		{"genesis", genesisPath, "", &genesisHash, false},
		{"metadata", "", metadataPath, &metadataHash, false},
		{"neither", "", "", nil, false},
		{"genesis precedence", genesisPath, "missing", &genesisHash, false},
		{"missing genesis", "missing", "", nil, true},
		{"missing metadata", "", "missing", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveNetworkHash(tc.genesis, tc.metadata)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v", err)
			}
			if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
				t.Fatalf("hash = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLoadGenesisConfigurationFileAcceptsMinimalDefaultCostModel(t *testing.T) {
	path := writeMinimalGenesisFile(t)

	genesis := loadGenesisConfigurationFile(context.Background(), path)

	if genesis.Storage_cost_model != expectedMinimalCostModel {
		t.Fatalf("unexpected storage cost model: %+v", genesis.Storage_cost_model)
	}
}

func TestSysCoordinatorCommandHasGenesisFileFlag(t *testing.T) {
	if flag := sysCoordinatorCmd.Flags().Lookup("genesis-file"); flag == nil {
		t.Fatal("sys coordinator is missing --genesis-file flag")
	}
}

func TestSysCoordinatorCommandHasHostedBootstrapFlags(t *testing.T) {
	for _, name := range []string{
		"hosted-bootstrap-metadata",
		"hosted-bootstrap-indexer",
		"hosted-bootstrap-scheduler-gateway",
		"hosted-bootstrap-db-gateway",
		"hosted-bootstrap-view-gateway",
		"hosted-bootstrap-public-rpc-url",
		"hosted-bootstrap-runtime-manifest-out",
	} {
		if flag := sysCoordinatorCmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("sys coordinator is missing --%s flag", name)
		}
	}
}

type fakeGenesisControl struct {
	tips        []proto.SchedulerGetTipResult
	getErr      error
	loadResults []proto.DbSchedResult
	loadErr     error
	loads       []proto.GenesisConfiguration
}

func (f *fakeGenesisControl) GetTip(context.Context) (proto.SchedulerGetTipResult, error) {
	if f.getErr != nil {
		return proto.SchedulerGetTipResult{}, f.getErr
	}
	if len(f.tips) == 0 {
		return proto.SchedulerGetTipResult{}, errors.New("unexpected GetTip call")
	}
	tip := f.tips[0]
	f.tips = f.tips[1:]
	return tip, nil
}

func (f *fakeGenesisControl) LoadGenesisConfiguration(_ context.Context, genesis proto.GenesisConfiguration) (proto.DbSchedResult, error) {
	f.loads = append(f.loads, genesis)
	if f.loadErr != nil {
		return 0, f.loadErr
	}
	if len(f.loadResults) == 0 {
		return 0, errors.New("unexpected LoadGenesisConfiguration call")
	}
	result := f.loadResults[0]
	f.loadResults = f.loadResults[1:]
	return result, nil
}

func tipOK(height uint64) proto.SchedulerGetTipResult {
	tip := proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipOK}
	tip.Db_tip().Value.Proposal_header.Block_height = proto.BlockHeight(height)
	return tip
}

func tipNoGenesis() proto.SchedulerGetTipResult {
	return proto.SchedulerGetTipResult{Outcome: proto.SchedulerGetTipNoGenesisBlock}
}

func TestLoadGenesisIfNeededSkipsWhenSchedulerAlreadyHasGenesis(t *testing.T) {
	fake := &fakeGenesisControl{tips: []proto.SchedulerGetTipResult{tipOK(7)}}

	loadGenesisIfNeeded(context.Background(), fake, writeMinimalGenesisFile(t))

	if len(fake.loads) != 0 {
		t.Fatalf("loaded genesis %d times, want 0", len(fake.loads))
	}
}

func TestLoadGenesisIfNeededLoadsMinimalGenesis(t *testing.T) {
	fake := &fakeGenesisControl{
		tips:        []proto.SchedulerGetTipResult{tipNoGenesis()},
		loadResults: []proto.DbSchedResult{proto.DbSchedResultOK},
	}

	loadGenesisIfNeeded(context.Background(), fake, writeMinimalGenesisFile(t))

	if len(fake.loads) != 1 {
		t.Fatalf("loaded genesis %d times, want 1", len(fake.loads))
	}
	if fake.loads[0].Storage_cost_model != expectedMinimalCostModel {
		t.Fatalf("unexpected loaded cost model: %+v", fake.loads[0].Storage_cost_model)
	}
}

func TestLoadGenesisIfNeededToleratesConcurrentGenesisLoad(t *testing.T) {
	fake := &fakeGenesisControl{
		tips: []proto.SchedulerGetTipResult{
			tipNoGenesis(),
			tipOK(1),
		},
		loadResults: []proto.DbSchedResult{proto.DbSchedResultError},
	}

	err := must.Try(func() {
		loadGenesisIfNeeded(context.Background(), fake, writeMinimalGenesisFile(t))
	})

	if err != nil {
		t.Fatalf("loadGenesisIfNeeded returned error for concurrent load: %v", err)
	}
	if len(fake.loads) != 1 {
		t.Fatalf("loaded genesis %d times, want 1", len(fake.loads))
	}
}

func TestLoadGenesisIfNeededFailsWhenLoadFailsAndTipStillHasNoGenesis(t *testing.T) {
	fake := &fakeGenesisControl{
		tips: []proto.SchedulerGetTipResult{
			tipNoGenesis(),
			tipNoGenesis(),
		},
		loadResults: []proto.DbSchedResult{proto.DbSchedResultError},
	}

	err := must.Try(func() {
		loadGenesisIfNeeded(context.Background(), fake, writeMinimalGenesisFile(t))
	})

	if err == nil {
		t.Fatal("loadGenesisIfNeeded succeeded, want error")
	}
	if !strings.Contains(err.Error(), "loading genesis failed") {
		t.Fatalf("error = %v, want loading genesis failure", err)
	}
}
