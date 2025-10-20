package storage

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
)

func TestWalkReverseDescendantsOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	touch := func(rel string) {
		t.Helper()
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(rel), 0o644); err != nil {
			t.Fatalf("write %q: %v", full, err)
		}
	}

	mkdir := func(rel string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", rel, err)
		}
	}

	// Build a small tree with files and empty directories.
	touch("alpha/a.txt")
	touch("alpha/zeta/deep.txt")
	mkdir("alpha/empty")
	touch("beta/file.txt")
	touch("beta/inner/omega.txt")
	mkdir("gamma")
	touch("root.txt")

	var visited []string
	if err := WalkReverseDescendants(context.Background(), root, func(path string, d fs.DirEntry) error {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		visited = append(visited, rel)
		return nil
	}); err != nil {
		t.Fatalf("WalkReverseDescendants returned error: %v", err)
	}

	expected := collectAllPaths(t, root)

	if !slices.Equal(visited, expected) {
		t.Fatalf("paths mismatch\nvisited:  %v\nexpected: %v", visited, expected)
	}
}

func TestWalkReverseDescendantsContextCancel(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, rel := range []string{"a.txt", "dir/b.txt"} {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %q: %v", full, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var visited []string
	err := WalkReverseDescendants(ctx, root, func(path string, d fs.DirEntry) error {
		visited = append(visited, path)
		cancel()
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	if len(visited) == 0 {
		t.Fatalf("expected at least one visited path before cancellation")
	}
}

func collectAllPaths(t *testing.T, root string) []string {
	t.Helper()

	var paths []string
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	}); err != nil {
		t.Fatalf("collect paths: %v", err)
	}

	sort.Slice(paths, func(i, j int) bool { return paths[i] > paths[j] })
	return paths
}
