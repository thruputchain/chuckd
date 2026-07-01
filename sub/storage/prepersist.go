package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
)

type PrePersist struct {
	lk   sync.Mutex
	cond *sync.Cond
	lks  map[string]bool
}

func NewPrePersist() *PrePersist {
	x := &PrePersist{lks: map[string]bool{}}
	x.cond = sync.NewCond(&x.lk)
	return x
}

func (x *PrePersist) lock(filepath string) {
	x.lk.Lock()
	defer x.lk.Unlock()
	for x.lks[filepath] {
		x.cond.Wait()
	}
	x.lks[filepath] = true
}

func (x *PrePersist) unlock(filepath string) {
	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()
	delete(x.lks, filepath)
}

// Load can return os.ErrNotExist, or a reader error from r.
func (x *PrePersist) Load(ctx context.Context, full string, r func(io.Reader) error) error {
	x.lock(full)
	defer x.unlock(full)

	prepersist := prepersist(ctx, full)

	if isPresentNonEmpty(ctx, prepersist) {
		liblog.Debugf(ctx, "loading pre-persisted %s", prepersist)
		file, err := os.Open(prepersist)
		must.NoError(ctx, err)
		defer file.Close()
		return r(file)
	}

	file, err := os.Open(full)
	if errors.Is(err, os.ErrNotExist) {
		return err
	}
	must.NoError(ctx, err)
	ensureNonEmpty(ctx, full)
	defer file.Close()

	return r(file)
}

// PrePersist is idempotent. If the persisted or pre-persisted file already exists, it does nothing.
func (x *PrePersist) PrePersist(ctx context.Context, full string, w func(io.Writer) error) {
	x.lock(full)
	defer x.unlock(full)

	prepersist := prepersist(ctx, full)

	liblog.Debugf(ctx, "pre-persisting %s to %s", full, prepersist)

	// if the target persist file already exists, do nothing
	if isPresentNonEmpty(ctx, full) {
		liblog.Debugf(ctx, "pre-persisting %s: persisted already", full)
		return
	}

	if isPresentNonEmpty(ctx, prepersist) { // if pre-persist copy exists, use it
		liblog.Debugf(ctx, "pre-persisting %s: pre-persisted already", prepersist)
		return
	}

	writeFileAtomically(ctx, prepersist, w)
}

// Persist is idempotent. If the file already exists, it does nothing.
// If a pre-persisted file exists, it is used.
func (x *PrePersist) Persist(ctx context.Context, full string, w func(io.Writer) error) {
	x.lock(full)
	defer x.unlock(full)

	prepersist := prepersist(ctx, full)

	liblog.Debugf(ctx, "persisting %s to %s", full, prepersist)

	if isPresentNonEmpty(ctx, full) {
		liblog.Debugf(ctx, "persisting %s: persisted already", full)
		return
	}

	if isPresentNonEmpty(ctx, prepersist) { // if pre-persist copy exists, use it
		liblog.Debugf(ctx, "persisting %s: using pre-persist copy", full)
		err := os.Rename(prepersist, full)
		must.NoError(ctx, err)
		syncDir(ctx, filepath.Dir(full))
		return
	}

	writeFileAtomically(ctx, full, w)
}

func (x *PrePersist) PersistAlreadyPrePersisted(ctx context.Context, full string) {
	x.lock(full)
	defer x.unlock(full)

	prepersist := prepersist(ctx, full)

	liblog.Debugf(ctx, "persisting already pre-persisted %s to %s", full, prepersist)

	if isPresentNonEmpty(ctx, full) {
		liblog.Debugf(ctx, "persisting already pre-persisted %s: persisted already", full)
		return
	}

	must.Assert(ctx, isPresentNonEmpty(ctx, prepersist))

	err := os.Rename(prepersist, full)
	must.NoError(ctx, err)
	syncDir(ctx, filepath.Dir(full))
}

// writeFileAtomically is idempotent. If the file already exists, it does nothing.
func writeFileAtomically(ctx context.Context, full string, w func(io.Writer) error) {

	if isPresentNonEmpty(ctx, full) { // if file exists, already done
		return
	}

	staging := staging(ctx, full)

	err := os.MkdirAll(filepath.Dir(staging), 0o755)
	must.NoError(ctx, err)

	file, err := os.OpenFile(staging, os.O_RDWR|os.O_CREATE, 0o644)
	must.NoError(ctx, err)

	err = w(file)
	must.NoError(ctx, err)

	// fsync the file's contents to disk before closing, so that the rename
	// below cannot expose a file with missing or partial data after a crash.
	err = file.Sync()
	must.NoError(ctx, err)

	err = file.Close()
	must.NoError(ctx, err)
	ensureNonEmpty(ctx, staging)

	// atomicly move staging to permanent
	err = os.Rename(staging, full)
	must.NoError(ctx, err)

	syncDir(ctx, filepath.Dir(full))
}

// syncDir fsyncs a directory so that a preceding rename into (or out of) it is
// durable. Without this, a crash could leave the directory entry in an
// undefined state even though the file's contents were already on disk.
func syncDir(ctx context.Context, dir string) {
	d, err := os.Open(dir)
	must.NoError(ctx, err)
	defer d.Close()
	err = d.Sync()
	must.NoError(ctx, err)
}

func prepersist(_ context.Context, full string) string {
	return path.Join(filepath.Dir(full), filepath.Base(full)+".prepersist")
}

func staging(ctx context.Context, full string) string {
	return path.Join(filepath.Dir(full), filepath.Base(full)+".staging-"+FilenameNonce(ctx))
}

func isPresent(full string) bool {
	_, err := os.Stat(full) // if file exists, already done
	if err == nil {
		return true
	}
	return false
}

func isPresentNonEmpty(ctx context.Context, full string) bool {
	info, err := os.Stat(full)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	must.NoError(ctx, err)
	if info.Size() == 0 {
		must.Panic(ctx, fmt.Errorf("persisted storage file is empty: %s", full))
	}
	return true
}

func ensureNonEmpty(ctx context.Context, full string) {
	if !isPresentNonEmpty(ctx, full) {
		must.Panic(ctx, fmt.Errorf("persisted storage file does not exist: %s", full))
	}
}
