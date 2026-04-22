package storage

import (
	"context"
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

func (x *PrePersist) Load(ctx context.Context, full string, hash [32]byte, r func(io.Reader) error) error {
	x.lock(full)
	defer x.unlock(full)

	pp := prepersistPath(full, hash)

	if isPresent(pp) {
		liblog.Debugf(ctx, "loading pre-persisted %s", pp)
		file, err := os.Open(pp)
		must.NoError(ctx, err)
		defer file.Close()
		return r(file)
	}

	file, err := os.Open(full)
	must.NoError(ctx, err)
	defer file.Close()

	return r(file)
}

// PrePersist is idempotent. If the persisted or pre-persisted file already exists, it does nothing.
// hash differentiates prepersist files when multiple values may target the same final path.
func (x *PrePersist) PrePersist(ctx context.Context, full string, hash [32]byte, w func(io.Writer) error) {
	x.lock(full)
	defer x.unlock(full)

	pp := prepersistPath(full, hash)

	liblog.Debugf(ctx, "pre-persisting %s to %s", full, pp)

	if isPresent(full) {
		liblog.Debugf(ctx, "pre-persisting %s: persisted already", full)
		return
	}

	if isPresent(pp) {
		liblog.Debugf(ctx, "pre-persisting %s: pre-persisted already", pp)
		return
	}

	writeFileAtomically(ctx, pp, w)
}

// Persist is idempotent. If the file already exists, it does nothing.
// If a pre-persisted file exists, it is used.
func (x *PrePersist) Persist(ctx context.Context, full string, hash [32]byte, w func(io.Writer) error) {
	x.lock(full)
	defer x.unlock(full)

	pp := prepersistPath(full, hash)

	liblog.Debugf(ctx, "persisting %s to %s", full, pp)

	if isPresent(full) {
		liblog.Debugf(ctx, "persisting %s: persisted already", full)
		return
	}

	if isPresent(pp) {
		liblog.Debugf(ctx, "persisting %s: using pre-persist copy", full)
		err := os.Rename(pp, full)
		must.NoError(ctx, err)
		return
	}

	writeFileAtomically(ctx, full, w)
}

func (x *PrePersist) PersistAlreadyPrePersisted(ctx context.Context, full string, hash [32]byte) {
	x.lock(full)
	defer x.unlock(full)

	pp := prepersistPath(full, hash)

	liblog.Debugf(ctx, "persisting already pre-persisted %s to %s", full, pp)

	if isPresent(full) {
		liblog.Debugf(ctx, "persisting already pre-persisted %s: persisted already", full)
		return
	}

	must.Assert(ctx, isPresent(pp))

	err := os.Rename(pp, full)
	must.NoError(ctx, err)
}

// writeFileAtomically is idempotent. If the file already exists, it does nothing.
func writeFileAtomically(ctx context.Context, full string, w func(io.Writer) error) {

	if isPresent(full) { // if file exists, already done
		return
	}

	staging := staging(ctx, full)

	err := os.MkdirAll(filepath.Dir(staging), 0o755)
	must.NoError(ctx, err)

	file, err := os.OpenFile(staging, os.O_RDWR|os.O_CREATE, 0o644)
	must.NoError(ctx, err)

	err = w(file)
	must.NoError(ctx, err)

	err = file.Close()
	must.NoError(ctx, err)

	// atomicly move staging to permanent
	err = os.Rename(staging, full)
	must.NoError(ctx, err)
}

func prepersistPath(full string, hash [32]byte) string {
	return path.Join(filepath.Dir(full), filepath.Base(full)+fmt.Sprintf(".prepersist-%x", hash))
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
