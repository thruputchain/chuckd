package librglob

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func mklink(t *testing.T, newname, oldname string) error {
	t.Helper()
	// Try to create a symlink; on Windows this may require developer mode/admin.
	_ = os.Remove(newname)
	return os.Symlink(oldname, newname)
}

func TestWalkReverse_BasicFiles(t *testing.T) {
	root := t.TempDir()

	// Layout:
	//  logs/2023/{a.json,c.json}
	//  logs/2024/{a.json,b.json}
	touch(t, filepath.Join(root, "logs", "2023", "a.json"))
	touch(t, filepath.Join(root, "logs", "2023", "c.json"))
	touch(t, filepath.Join(root, "logs", "2024", "a.json"))
	touch(t, filepath.Join(root, "logs", "2024", "b.json"))

	pat := filepath.Join(root, "logs", "*", "*.json")

	var got []string
	err := WalkReverse(pat, func(p string) error {
		got = append(got, filepath.Clean(p))
		return nil
	}, &Options{})
	if err != nil {
		t.Fatalf("WalkReverse: %v", err)
	}

	// Expected reverse lexicographic order of full paths.
	exp := []string{
		filepath.Join(root, "logs", "2024", "b.json"),
		filepath.Join(root, "logs", "2024", "a.json"),
		filepath.Join(root, "logs", "2023", "c.json"),
		filepath.Join(root, "logs", "2023", "a.json"),
	}
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("mismatch.\n got:\n  %s\nexp:\n  %s", strings.Join(got, "\n  "), strings.Join(exp, "\n  "))
	}
}

func TestWalkReverse_IncludeDirsAtLeaf(t *testing.T) {
	root := t.TempDir()

	// data/{x,y}/(files)
	os.MkdirAll(filepath.Join(root, "data", "x"), 0o755)
	os.MkdirAll(filepath.Join(root, "data", "y"), 0o755)
	touch(t, filepath.Join(root, "data", "x", "file.txt"))
	touch(t, filepath.Join(root, "data", "y", "alpha.txt"))

	pat := filepath.Join(root, "data", "*")

	var got []string
	err := WalkReverse(pat, func(p string) error {
		got = append(got, filepath.Clean(p))
		return nil
	}, &Options{IncludeDirs: true})
	if err != nil {
		t.Fatalf("WalkReverse: %v", err)
	}

	// Expect only directories, reverse lex by name ("y" then "x").
	exp := []string{
		filepath.Join(root, "data", "y"),
		filepath.Join(root, "data", "x"),
	}
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("mismatch.\n got:\n  %s\nexp:\n  %s", strings.Join(got, "\n  "), strings.Join(exp, "\n  "))
	}
}

func TestWalkReverse_DoubleStarTxt(t *testing.T) {
	root := t.TempDir()

	// Create a small tree with multiple .txt in various dirs.
	touch(t, filepath.Join(root, "top.txt"))
	touch(t, filepath.Join(root, "dirB", "sub", "z.txt"))
	touch(t, filepath.Join(root, "data", "x", "file.txt"))
	touch(t, filepath.Join(root, "data", "y", "alpha.txt"))
	touch(t, filepath.Join(root, "data", "y", "beta.txt"))

	pat := filepath.Join(root, "**", "*.txt")

	var got []string
	err := WalkReverse(pat, func(p string) error {
		got = append(got, filepath.Clean(p))
		return nil
	}, &Options{DoubleStar: true})
	if err != nil {
		t.Fatalf("WalkReverse: %v", err)
	}

	// Compute expected (reverse lex) from known set.
	exp := []string{
		filepath.Join(root, "top.txt"),
		filepath.Join(root, "dirB", "sub", "z.txt"),
		filepath.Join(root, "data", "y", "beta.txt"),
		filepath.Join(root, "data", "y", "alpha.txt"),
		filepath.Join(root, "data", "x", "file.txt"),
	}

	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("mismatch.\n got:\n  %s\nexp:\n  %s", strings.Join(got, "\n  "), strings.Join(exp, "\n  "))
	}
}

func TestWalkReverse_NonMetaPath(t *testing.T) {
	root := t.TempDir()

	// A file and a directory with no meta in pattern.
	touch(t, filepath.Join(root, "plain.txt"))
	if err := os.MkdirAll(filepath.Join(root, "adir"), 0o755); err != nil {
		t.Fatal(err)
	}

	// 1) file path should emit file.
	{
		pat := filepath.Join(root, "plain.txt")
		var got []string
		err := WalkReverse(pat, func(p string) error {
			got = append(got, p)
			return nil
		}, nil)
		if err != nil {
			t.Fatalf("WalkReverse: %v", err)
		}
		exp := []string{filepath.Join(root, "plain.txt")}
		if !reflect.DeepEqual(got, exp) {
			t.Fatalf("file mismatch: got %v exp %v", got, exp)
		}
	}

	// 2) directory path should NOT emit when IncludeDirs is false.
	{
		pat := filepath.Join(root, "adir")
		var got []string
		err := WalkReverse(pat, func(p string) error {
			got = append(got, p)
			return nil
		}, &Options{IncludeDirs: false})
		if err != nil {
			t.Fatalf("WalkReverse: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected no output for dir without IncludeDirs, got %v", got)
		}
	}

	// 3) directory path SHOULD emit when IncludeDirs is true.
	{
		pat := filepath.Join(root, "adir")
		var got []string
		err := WalkReverse(pat, func(p string) error {
			got = append(got, p)
			return nil
		}, &Options{IncludeDirs: true})
		if err != nil {
			t.Fatalf("WalkReverse: %v", err)
		}
		exp := []string{filepath.Join(root, "adir")}
		if !reflect.DeepEqual(got, exp) {
			t.Fatalf("dir mismatch: got %v exp %v", got, exp)
		}
	}
}

// XXX
func TestWalkReverse_SymlinkFollow(t *testing.T) {
	// Symlinks are finicky on Windows; skip if not supported.
	if runtime.GOOS == "windows" {
		t.Skip("symlink test skipped on Windows")
	}
	root := t.TempDir()

	// data/y with two txt files
	touch(t, filepath.Join(root, "data", "y", "alpha.txt"))
	touch(t, filepath.Join(root, "data", "y", "beta.txt"))

	// link_to_y -> data/y
	link := filepath.Join(root, "link_to_y")
	if err := mklink(t, link, filepath.Join(root, "data", "y")); err != nil {
		t.Skipf("symlink creation failed, skipping: %v", err)
	}

	// Pattern through the symlink
	pat := filepath.Join(link, "*.txt")

	var got []string
	// 1) FollowSymlinks=false => should still match files via the link (the link
	// itself is the base we start from; traversal decisions only matter for descents).
	err := WalkReverse(pat, func(p string) error {
		got = append(got, filepath.Clean(p))
		return nil
	}, &Options{FollowSymlinks: false})
	if err != nil {
		t.Fatalf("WalkReverse: %v", err)
	}
	exp := []string{
		filepath.Join(link, "beta.txt"),
		filepath.Join(link, "alpha.txt"),
	}
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("no-follow mismatch.\n got:\n  %s\nexp:\n  %s", strings.Join(got, "\n  "), strings.Join(exp, "\n  "))
	}

	// 2) Confirm that following symlinked dirs during descent also works:
	// link_to_y/* (same result here because depth is only one).
	got = nil
	err = WalkReverse(pat, func(p string) error {
		got = append(got, filepath.Clean(p))
		return nil
	}, &Options{FollowSymlinks: true})
	if err != nil {
		t.Fatalf("WalkReverse: %v", err)
	}
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("follow mismatch.\n got:\n  %s\nexp:\n  %s", strings.Join(got, "\n  "), strings.Join(exp, "\n  "))
	}
}

func TestWalkReverse_InvalidPattern(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "logs", "a.json"))
	pat := filepath.Join(root, "logs", "[*", "*.json") // invalid char class

	var gotErr error
	err := WalkReverse(pat, func(string) error { return nil }, nil)
	if err != nil {
		gotErr = err
	}
	if gotErr == nil {
		t.Fatalf("expected an error for invalid pattern, got nil")
	}
	// filepath.Match returns "syntax error in pattern"
	if !strings.Contains(strings.ToLower(gotErr.Error()), "syntax") {
		// Accept any error that's clearly coming from pattern parsing.
		t.Fatalf("unexpected error: %v", gotErr)
	}
}

func TestWalkReverse_CallbackErrorPropagates(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "a.txt"))
	touch(t, filepath.Join(root, "b.txt"))

	pat := filepath.Join(root, "*.txt")

	want := errors.New("stop")
	var count int
	err := WalkReverse(pat, func(string) error {
		count++
		return want
	}, nil)
	if !errors.Is(err, want) {
		t.Fatalf("expected error to propagate: got %v", err)
	}
	if count != 1 {
		t.Fatalf("callback should be called once before error; got %d", count)
	}
}

// XXX
func TestWalkReverse_GlobalReverseOrderProperty(t *testing.T) {
	// Sanity check that the emitted paths are reverse-lex order
	// compared to the set of matches, for a richer tree.
	root := t.TempDir()

	paths := []string{
		filepath.Join(root, "a", "a.txt"),
		filepath.Join(root, "a", "b.txt"),
		filepath.Join(root, "b", "a.txt"),
		filepath.Join(root, "b", "b.txt"),
		filepath.Join(root, "b", "ba", "c.txt"),
		filepath.Join(root, "c.txt"),
	}
	for _, p := range paths {
		touch(t, p)
	}
	pat := filepath.Join(root, "**", "*.txt")

	var got []string
	if err := WalkReverse(pat, func(p string) error {
		got = append(got, filepath.Clean(p))
		return nil
	}, &Options{DoubleStar: true}); err != nil {
		t.Fatalf("WalkReverse: %v", err)
	}

	// Compute canonical expected by gathering all, sorting ascending, reversing.
	exp := append([]string(nil), got...) // placeholder size
	exp = nil
	// Recreate set from layout:
	exp = append(exp,
		filepath.Join(root, "a", "a.txt"),
		filepath.Join(root, "a", "b.txt"),
		filepath.Join(root, "b", "a.txt"),
		filepath.Join(root, "b", "b.txt"),
		filepath.Join(root, "b", "ba", "c.txt"),
		filepath.Join(root, "c.txt"),
	)
	sort.Strings(exp)
	for i, j := 0, len(exp)-1; i < j; i, j = i+1, j-1 {
		exp[i], exp[j] = exp[j], exp[i]
	}

	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("reverse-order property mismatch.\n got:\n  %s\nexp:\n  %s", strings.Join(got, "\n  "), strings.Join(exp, "\n  "))
	}
}
