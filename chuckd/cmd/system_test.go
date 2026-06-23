package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestLoadGenesisConfigurationFileAcceptsMinimalDefaultCostModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "genesis.json")
	data := []byte(`{
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
}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	genesis := loadGenesisConfigurationFile(context.Background(), path)

	if len(genesis.Initial_config) != 0 {
		t.Fatalf("initial config length = %d, want 0", len(genesis.Initial_config))
	}
	if genesis.Storage_cost_model != (proto.StorageCostModelXdr{
		Base_data_item_cost:               12000,
		Bytestring_cost_per_byte:          10,
		Base_contract_deployment_cost:     12000,
		Contract_deployment_cost_per_byte: 10,
		Auth_per_key_cost:                 1111,
		Rcq_cost_per_entry:                8,
		Replay_cache_cost_per_entry:       7,
		Balance_cost:                      1234,
	}) {
		t.Fatalf("unexpected storage cost model: %+v", genesis.Storage_cost_model)
	}
}
