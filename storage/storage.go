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
	//
	ready chan struct{}
}

func NewStorage(ctx context.Context, cfg Config) *DiskStorage {
	return &DiskStorage{
		cfg:      cfg,
		catchupM: xdrutil.NewXdrMarshaler(),
		scpM:     xdrutil.NewXdrMarshaler(),
		ready:    make(chan struct{}),
	}
}

func (x *DiskStorage) Start(ctx context.Context, dir string) {
	defer close(x.ready)

	x.dir = dir

	// obtain an advisory lock
	err := os.MkdirAll(dir, 0o755)
	must.NoError(ctx, err)

	flk := flock.New(path.Join(dir, "lock"))
	_, err = flk.TryLock()
	must.NoError(ctx, err)

	go x.reclaimLoop(ctx)
}

// catchup blocks

func (x *DiskStorage) PersistCatchupBlock(ctx context.Context, cb proto.CatchupBlock) {
	<-x.ready

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
	<-x.ready

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

func (x *DiskStorage) PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	<-x.ready

	// lock ensures concurrent duplicate adds don't interfere
	x.consensusLk.Lock()
	defer x.consensusLk.Unlock()

	full := x.ConsensusValueFilepath(cv.Nomination.Block_height)
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

func (x *DiskStorage) FindConsensusValue(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue {
	<-x.ready

	full := x.ConsensusValueFilepath(height)

	bytes, err := os.ReadFile(full)
	if err != nil {
		return nil
	}

	cv := &proto.ConsensusValue{}
	err = xdrutil.UnmarshalXdr(cv, bytes)
	must.NoError(ctx, err)

	return cv
}

func (x *DiskStorage) findConsensusValueMaxHeight(ctx context.Context) proto.BlockHeight {
	<-x.ready

	rootpath := x.ConsensusValuesDirpath()
	err := os.MkdirAll(filepath.Dir(rootpath), 0o755)
	must.NoError(ctx, err)

	var maxHeight int64

	visit := func(path string, d fs.DirEntry) error {
		if d.IsDir() {
			return nil // skip dirs
		}
		height, err := ParseConsensusValueFilename(d.Name())
		if err != nil {
			return nil // skip non-consensus files
		}
		maxHeight = height
		return errStopWalking
	}

	err = WalkReverseDescendants(ctx, rootpath, visit)
	must.Assert(ctx, err == nil || errors.Is(err, errStopWalking))

	return maxHeight
}

var errStopWalking = fmt.Errorf("stop reverse file walk")

func (x *DiskStorage) FindConsensusValuesAbove(ctx context.Context, above proto.BlockHeight) []proto.ConsensusValue {
	found := []proto.ConsensusValue{}
	maxHeight := x.findConsensusValueMaxHeight(ctx)
	for height := above + 1; height <= maxHeight; height++ {
		cv := x.FindConsensusValue(ctx, height)
		if cv == nil {
			continue
		}
		found = append(found, *cv)
	}
	return found
}

// scp envelopes

func (x *DiskStorage) PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.ready

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
	<-x.ready

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
