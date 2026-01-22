package storage

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

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

func (x *PrePersist) Load(ctx context.Context, full string, r func(io.Reader) error) error {
	x.lock(full)
	defer x.unlock(full)

	file, err := os.Open(full)
	must.NoError(ctx, err)

	return r(file)
}

func (x *PrePersist) PrePersist(ctx context.Context, full string, w func(io.Writer) error) {
	x.lock(full)
	defer x.unlock(full)

	prepersist := prepersist(ctx, full)
	writeFileAtomically(ctx, prepersist, w)
}

func (x *PrePersist) Persist(ctx context.Context, full string, w func(io.Writer) error) {
	x.lock(full)
	defer x.unlock(full)

	prepersist := prepersist(ctx, full)
	if isPresent(prepersist) { // if pre-persist copy exists, use it
		err := os.Rename(prepersist, full)
		must.NoError(ctx, err)
		return
	}

	writeFileAtomically(ctx, full, w)
}

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

func prepersist(ctx context.Context, full string) string {
	return path.Join(filepath.Dir(full), filepath.Base(full)+".prepersist-"+FilenameNonce(ctx))
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
