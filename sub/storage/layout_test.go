package storage

import (
	"os"
	"path/filepath"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
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

func mustHash(fill byte) proto.ConsensusValueHash {
	var h proto.ConsensusValueHash
	for i := range h {
		h[i] = fill
		fill++
	}
	return h
}

func TestFindCatchupFilesByHash_Persisted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	h := mustHash(0x01)
	full := filepath.Join(dir, "catchup-000000000123-"+toHex(h)+".xdr")
	touch(t, full)

	got, err := FindCatchupFilesByHash(dir, h)
	if err != nil {
		t.Fatalf("FindCatchupFilesByHash returned error: %v", err)
	}
	if got != full {
		t.Fatalf("got %q, want %q", got, full)
	}
}

func TestFindCatchupFilesByHash_PrePersisted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	h := mustHash(0x11)
	full := filepath.Join(dir, "catchup-000000000777-"+toHex(h)+".xdr.prepersist")
	touch(t, full)

	got, err := FindCatchupFilesByHash(dir, h)
	if err != nil {
		t.Fatalf("FindCatchupFilesByHash returned error: %v", err)
	}
	if got != full {
		t.Fatalf("got %q, want %q", got, full)
	}
}

func TestFindCatchupFilesByHash_NotFoundReturnsEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	h := mustHash(0x21)

	got, err := FindCatchupFilesByHash(dir, h)
	if err != nil {
		t.Fatalf("FindCatchupFilesByHash returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestFindCatchupFilesByHash_MultipleMatchesError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	h := mustHash(0x31)
	touch(t, filepath.Join(dir, "catchup-000000000001-"+toHex(h)+".xdr"))
	touch(t, filepath.Join(dir, "catchup-000000000002-"+toHex(h)+".xdr"))

	got, err := FindCatchupFilesByHash(dir, h)
	if err == nil {
		t.Fatalf("expected error, got nil and path %q", got)
	}
}

func TestFindCatchupFilesByHash_MissingDirReturnsEmpty(t *testing.T) {
	t.Parallel()

	h := mustHash(0x41)
	got, err := FindCatchupFilesByHash(filepath.Join(t.TempDir(), "missing"), h)
	if err != nil {
		t.Fatalf("FindCatchupFilesByHash returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func toHex(h proto.ConsensusValueHash) string {
	return stringLowerHex(h[:])
}

func stringLowerHex(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}
