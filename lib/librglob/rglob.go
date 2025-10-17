// Package rglob provides reverse-lexicographic glob walking.
package librglob

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Options struct {
	// Emit directories that fully match the pattern (in addition to files).
	IncludeDirs bool
	// Follow directory symlinks. Off by default to avoid cycles.
	FollowSymlinks bool
	// Support the "**" segment to match zero or more directories.
	DoubleStar bool
}

// WalkReverse walks filesystem entries matching the glob pattern in reverse
// lexicographic order and calls fn(path) for each match. The traversal is
// streamed: it does NOT build and reverse-sort a giant list first.
func WalkReverse(pattern string, fn func(string) error, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	clean := filepath.Clean(pattern)

	// Split into segments and locate the static base (no meta) prefix.
	segs := splitPath(clean)
	baseIdx := firstMetaIndex(segs)
	var base string
	switch {
	case baseIdx == -1:
		// No meta at all: just test existence and (optionally) directory inclusion.
		info, err := os.Lstat(clean)
		if err != nil {
			// No matches.
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		// If it's a dir and IncludeDirs=false, treat as non-match unless it’s a file.
		if info.IsDir() && !opts.IncludeDirs {
			return nil
		}
		return fn(clean)

	case baseIdx == 0:
		base = string(filepath.Separator)
		// On Unix, Clean("/") == "/" and splitPath("/") gives ["/"] – handle below.
		if isRoot(segs[0]) {
			// pattern like "/a*/b*.txt"
			return globInto(base, segs[1:], fn, opts)
		}
	default:
		// Join the non-meta prefix
		base = joinSegments(segs[:baseIdx])
		// Ensure base exists and is a dir we can descend into.
		if st, err := os.Lstat(base); err != nil {
			// Nothing to match below a missing base.
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		} else if !st.IsDir() {
			// If the base is a file but the pattern continues — no matches.
			return nil
		}
	}

	// Descend from base using the remaining (meta) segments.
	return globInto(base, segs[baseIdx:], fn, opts)
}

// ---- helpers ----

func globInto(dir string, segs []string, fn func(string) error, opts *Options) error {
	// Terminal: consumed all segments -> emit dir if allowed.
	if len(segs) == 0 {
		// Only emit existing leaf path (dir or file) depending on IncludeDirs.
		info, err := os.Lstat(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if info.IsDir() && !opts.IncludeDirs {
			return nil
		}
		return fn(filepath.Clean(dir))
	}

	seg := segs[0]

	// Handle "**" (if enabled): match zero or more directories.
	if opts.DoubleStar && seg == "**" {
		// 1) Match zero directories: move to next segment at same dir.
		if err := globInto(dir, segs[1:], fn, opts); err != nil {
			return err
		}
		// 2) Match one-or-more directories: descend into all subdirs (reverse order),
		// then keep the "**" in place to allow further depth.
		entries, err := readDirReverse(dir)
		if err != nil {
			// Treat unreadable directories as non-fatal unless it's the root target.
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		for _, de := range entries {
			if !de.IsDir() && !(opts.FollowSymlinks && isDirSymlink(dir, de)) {
				continue
			}
			next := filepath.Join(dir, de.Name())
			if err := globInto(next, segs, fn, opts); err != nil {
				return err
			}
		}
		return nil
	}

	// Normal segment: filter names in this dir by filepath.Match(seg, name)
	entries, err := readDirReverse(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	for _, de := range entries {
		name := de.Name()
		ok, matchErr := filepath.Match(seg, name)
		if matchErr != nil {
			// Invalid pattern segment -> surfacing error mimics filepath.Glob behavior.
			return matchErr
		}
		if !ok {
			continue
		}

		path := filepath.Join(dir, name)
		isDir := de.IsDir() || (opts.FollowSymlinks && isDirSymlink(dir, de))

		// If this is the last segment, we can emit files immediately (and dirs if requested).
		if len(segs) == 1 {
			if isDir {
				if opts.IncludeDirs {
					if err := fn(path); err != nil {
						return err
					}
				}
				continue
			}
			// It’s a file (or non-dir symlink): emit.
			if err := fn(path); err != nil {
				return err
			}
			continue
		}

		// Not the last segment: only descend into directories.
		if isDir {
			if err := globInto(path, segs[1:], fn, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

func readDirReverse(dir string) ([]fs.DirEntry, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// Reverse lexicographic by name.
	sort.Slice(ents, func(i, j int) bool {
		return ents[i].Name() > ents[j].Name()
	})
	return ents, nil
}

func isDirSymlink(parent string, de fs.DirEntry) bool {
	if de.Type()&os.ModeSymlink == 0 {
		return false
	}
	fi, err := os.Stat(filepath.Join(parent, de.Name()))
	return err == nil && fi.IsDir()
}

func splitPath(p string) []string {
	if p == "" {
		return []string{"."}
	}
	// Special-case root on Unix / volume roots on Windows.
	if isRoot(p) {
		return []string{p}
	}
	var segs []string
	for {
		dir, file := filepath.Split(p)
		if dir == "" {
			// last component
			segs = append([]string{file}, segs...)
			break
		}
		trimmed := strings.TrimRight(dir, string(filepath.Separator))
		if trimmed == "" && strings.HasPrefix(dir, string(filepath.Separator)) {
			// Reached root like "/"
			if file != "" {
				segs = append([]string{file}, segs...)
			}
			segs = append([]string{string(filepath.Separator)}, segs...)
			break
		}
		segs = append([]string{file}, segs...)
		p = trimmed
	}
	return segs
}

func joinSegments(segs []string) string {
	if len(segs) == 0 {
		return "."
	}
	// Preserve leading root if present.
	if len(segs) > 0 && isRoot(segs[0]) {
		return filepath.Join(segs...)
	}
	return filepath.Join(segs...)
}

func isRoot(s string) bool {
	// Unix "/"
	if s == string(filepath.Separator) {
		return true
	}
	// Windows volume roots: "C:\" or "C:/"
	if len(s) >= 2 && s[1] == ':' {
		clean := filepath.Clean(s)
		// After Clean, volume root becomes "C:\"
		return strings.HasSuffix(clean, ":\\") || strings.HasSuffix(clean, ":/")
	}
	return false
}

func firstMetaIndex(segs []string) int {
	for i, s := range segs {
		// Skip root segment
		if i == 0 && isRoot(s) {
			continue
		}
		if hasMeta(s) {
			return i
		}
	}
	return -1
}

func hasMeta(s string) bool {
	// filepath.Match meta: *, ?, [class]
	// Note: "**" is handled explicitly as a segment when opts.DoubleStar is true.
	return strings.ContainsAny(s, "*?[")
}
