package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

const minimalGenesisJSON = `{
  "initial_config": [],
  "storage_cost_model": {
    "base_data_item_cost": 12000,
    "bytestring_cost_per_byte": 10,
    "base_contract_deployment_cost": 12000,
    "contract_deployment_cost_per_byte": 10,
    "auth_per_key_cost": 1111,
    "rcq_cost_per_entry": 8,
    "replay_cache_cost_per_entry": 7,
    "balance_cost": 1234
  }
}`

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
	path := filepath.Join(t.TempDir(), "genesis.json")
	data := []byte(minimalGenesisJSON)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadGenesisConfigurationFileAcceptsMinimalDefaultCostModel(t *testing.T) {
	path := writeMinimalGenesisFile(t)

	genesis := loadGenesisConfigurationFile(context.Background(), path)

	if len(genesis.Initial_config) != 0 {
		t.Fatalf("initial config length = %d, want 0", len(genesis.Initial_config))
	}
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
	tip.Db_tip().Height = proto.BlockHeight(height)
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
