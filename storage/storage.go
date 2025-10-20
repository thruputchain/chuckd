package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type DiskStorage struct {
	dir string
	cfg Config
	//
	catchupLk   sync.Mutex
	scpLk       sync.Mutex
	consensusLk sync.Mutex
	//
	catchupM xdrutil.XdrMarshaler
	scpM     xdrutil.XdrMarshaler
}

func NewStorage(ctx context.Context, dir string, cfg Config) *DiskStorage {

	// obtain an advisory lock
	err := os.MkdirAll(dir, 0o755)
	must.NoError(ctx, err)

	flk := flock.New(path.Join(dir, "lock"))
	_, err = flk.TryLock()
	must.NoError(ctx, err)

	x := &DiskStorage{
		dir:      dir,
		cfg:      cfg,
		catchupM: xdrutil.NewXdrMarshaler(),
		scpM:     xdrutil.NewXdrMarshaler(),
	}

	go x.reclaimLoop(ctx)
	return x
}

// catchup blocks

func (x *DiskStorage) AddCatchupBlock(ctx context.Context, cb proto.CatchupBlock) {
	// lock ensures concurrent duplicate adds don't interfere
	// no lock is necessary on reading or garbage collection
	x.catchupLk.Lock()
	defer x.catchupLk.Unlock()

	full := x.CatchupBlockFilepath(cb.Consensus_value.Nomination.Block_height)
	fullStaging := path.Join(filepath.Dir(full), filepath.Base(full)+".staging-"+FilenameNonce(ctx))

	err := os.MkdirAll(filepath.Dir(full), 0o755)
	must.NoError(ctx, err)

	// write staging file
	borrowBytes := x.catchupM.MarshalXdr(&cb)
	defer borrowBytes.Recycle()

	err = os.WriteFile(fullStaging, borrowBytes.Bytes(), 0o644)
	must.NoError(ctx, err)

	// atomicly move staging to permanent
	err = os.Rename(fullStaging, full)
	must.NoError(ctx, err)
}

func (x *DiskStorage) FindCatchupBlock(ctx context.Context, height proto.BlockHeight) (proto.CatchupBlock, error) {

	full := x.CatchupBlockFilepath(height)

	bytes, err := os.ReadFile(full)
	if err != nil {
		return proto.CatchupBlock{}, err
	}

	cb := &proto.CatchupBlock{}
	err = xdrutil.UnmarshalXdr(cb, bytes)
	if err != nil {
		return proto.CatchupBlock{}, err
	}

	return *cb, nil
}

// consensus values

func (x *DiskStorage) AddConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	// lock ensures concurrent duplicate adds don't interfere
	x.consensusLk.Lock()
	defer x.consensusLk.Unlock()

	stamp := protoutil.TimeFromTimepoint(cv.Timestamp)
	full := x.ConsensusValueFilepath(stamp, cv.Nomination.Block_height)
	fullStaging := path.Join(filepath.Dir(full), filepath.Base(full)+".staging-"+FilenameNonce(ctx))

	err := os.MkdirAll(filepath.Dir(full), 0o755)
	must.NoError(ctx, err)

	// write staging file
	borrowBytes := x.catchupM.MarshalXdr(&cv)
	defer borrowBytes.Recycle()

	err = os.WriteFile(fullStaging, borrowBytes.Bytes(), 0o644)
	must.NoError(ctx, err)

	// atomicly move staging to permanent
	err = os.Rename(fullStaging, full)
	must.NoError(ctx, err)
}

var errStopWalking = fmt.Errorf("stop reverse file walk")

func (x *DiskStorage) FindLatestConsensusValues(ctx context.Context, n int) ([]proto.ConsensusValue, error) {

	rootpath := x.ConsensusValuesDirpath()
	err := os.MkdirAll(filepath.Dir(rootpath), 0o755)
	must.NoError(ctx, err)

	paths := make([]string, 0, n)

	visit := func(path string, d fs.DirEntry) error {
		if len(paths) >= cap(paths) {
			return errStopWalking
		}
		if d.IsDir() {
			return nil // skip dirs
		}
		_, err := ParseConsensusValueFilename(d.Name())
		if err != nil {
			return nil // skip non-consensus files
		}
		paths = append(paths, path)
		return nil
	}

	err = WalkReverseDescendants(ctx, rootpath, visit)
	must.Assert(ctx, err == nil || errors.Is(err, errStopWalking))

	// decode values
	cvs := make([]proto.ConsensusValue, len(paths))
	for i, path := range paths {

		bytes, err := os.ReadFile(path)
		must.NoError(ctx, err)

		err = xdrutil.UnmarshalXdr(&cvs[i], bytes)
		must.NoError(ctx, err)

	}
	return cvs, nil
}

// scp envelopes

func (x *DiskStorage) AddSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	// lock ensures concurrent duplicate adds don't interfere
	// note that no lock is necessary when reclaiming or reading
	x.scpLk.Lock()
	defer x.scpLk.Unlock()

	hash := xdrutil.HashXdr(x.scpM, &env)

	full := x.SCPEnvelopeFilepath(env.Statement.Slot_index, time.Now(), hash[:5])
	fullStaging := path.Join(filepath.Dir(full), filepath.Base(full)+".staging-"+FilenameNonce(ctx))

	err := os.MkdirAll(filepath.Dir(full), 0o755)
	must.NoError(ctx, err)

	// write staging file
	borrowBytes := x.scpM.MarshalXdr(&env)
	defer borrowBytes.Recycle()

	err = os.WriteFile(fullStaging, borrowBytes.Bytes(), 0o644)
	must.NoError(ctx, err)

	// atomicly move staging to permanent
	err = os.Rename(fullStaging, full)
	must.NoError(ctx, err)

}

func (x *DiskStorage) FindSCPEnvelopesSinceSlot(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error) {

	found := []proto.SCPEnvelope{}

	for {
		slotpath := x.SCPEnvelopeSlotpath(slot)
		slot += 1

		envpaths, err := ListScpFiles(slotpath)
		if err != nil {
			break
		}

		for _, envpath := range envpaths {
			env := proto.SCPEnvelope{}

			bytes, err := os.ReadFile(envpath)
			must.NoError(ctx, err)

			xdrutil.UnmarshalXdr(&env, bytes)
			must.NoError(ctx, err)

			found = append(found, env)
		}
	}

	return found, nil
}
