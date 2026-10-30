package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thruputchain/chuckd/util/configtext"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// config init writes the operator representation that LoadCoordinatorConfig
// reads, into a file (and directory) only this user can read, and never over
// an existing file.
func TestConfigInitWritesPrivateOperatorConfig(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".chuckd")
	cfgPath := filepath.Join(dir, "chuckd.json")
	rootConfigPath = cfgPath
	t.Cleanup(func() { rootConfigPath = "" })

	if err := configInitCmd.RunE(configInitCmd, nil); err != nil {
		t.Fatal(err)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("config directory mode %o, want 700", perm)
	}
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file mode %o, want 600", perm)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg proto.Config
	if err := configtext.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("config init output is not in the operator format: %v", err)
	}
	if cfg.Validator_config == nil || cfg.Validator_config.Private_info.Secret_key.Scheme != proto.Ed25519 {
		t.Fatal("config init did not write an Ed25519 identity")
	}

	if err := configInitCmd.RunE(configInitCmd, nil); err == nil {
		t.Fatal("config init replaced an existing config")
	}
}
