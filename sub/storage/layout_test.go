package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, full string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %q: %v", full, err)
	}
}
