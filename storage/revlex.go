package storage

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// WalkReverseDescendants walks all descendants of root (excluding root itself)
// in descending lexicographic order of their full paths. It calls visit for each
// path. Returning any error stops the walk.
func WalkReverseDescendants(
	ctx context.Context,
	root string,
	visit func(path string, d fs.DirEntry) error,
) error {
	type item struct {
		dir      string      // parent directory
		entry    fs.DirEntry // child entry under dir
		expanded bool        // for directories: whether children were already pushed
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	// Sort by base name descending so larger prefixes get processed first.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() > entries[j].Name() })

	// Pre-size stack approximately; avoids deep recursion.
	st := make([]item, 0, len(entries)+16)
	for _, e := range entries {
		st = append(st, item{dir: root, entry: e})
	}

	for len(st) > 0 {
		it := st[len(st)-1]
		st = st[:len(st)-1]

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		full := filepath.Join(it.dir, it.entry.Name())

		if it.entry.IsDir() && !it.expanded {
			kids, err := os.ReadDir(full)
			if err != nil {
				return err
			}
			sort.Slice(kids, func(i, j int) bool { return kids[i].Name() < kids[j].Name() })

			st = append(st, item{dir: it.dir, entry: it.entry, expanded: true})
			for _, kid := range kids {
				st = append(st, item{dir: full, entry: kid})
			}
			continue
		}

		if err := visit(full, it.entry); err != nil {
			return err
		}
	}

	return nil
}
