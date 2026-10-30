package protoutil

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	proto "github.com/thruputchain/thruput-protocol-go"
)

// config group's files hold secret keys: they are created readable by this
// user only, and a file left by an older version with wider permissions is
// refused rather than refilled with a new key behind its old mode.
func TestWriteConfigsToDirKeepsSecretsPrivate(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "group")
	cfgs := []*proto.Config{{Validator_config: InitConfig(ctx, rand.Reader, "")}}

	WriteConfigsToDir(ctx, cfgs, dir)
	for _, p := range []struct {
		path string
		want os.FileMode
	}{{dir, 0o700}, {filepath.Join(dir, "config-0.json"), 0o600}} {
		info, err := os.Stat(p.path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != p.want {
			t.Errorf("%s has mode %o, want %o", p.path, perm, p.want)
		}
	}

	// The upgrade case: a config written by the previous version.
	old := filepath.Join(dir, "config-1.json")
	if err := os.WriteFile(old, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := WriteNewConfigFile(old, []byte("new secret"))
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing config was not refused: %v", err)
	}
	data, err := os.ReadFile(old)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("existing config was overwritten: %q", data)
	}
}
