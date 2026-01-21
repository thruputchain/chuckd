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
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
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
	catchupM libxdr.XdrMarshaler
	scpM     libxdr.XdrMarshaler
	//
	ready chan struct{}
}

func NewStorage(ctx context.Context, cfg Config) *DiskStorage {
	return &DiskStorage{
		cfg:      cfg,
		catchupM: libxdr.NewXdrMarshaler(),
		scpM:     libxdr.NewXdrMarshaler(),
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

func (x *DiskStorage) FindCatchupBlock(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock {
	<-x.ready

	full := x.CatchupBlockFilepath(height)

	bytes, err := os.ReadFile(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	must.NoError(ctx, err)

	cb := &proto.CatchupBlock{}
	err = libxdr.UnmarshalXdr(cb, bytes)
	must.NoError(ctx, err)

	return cb
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

	//
	t0 := time.Now()

	file, err := os.Open(fullStaging)
	must.NoError(ctx, err)

	libxdr.MarshalXdrWriter(file, &cv)

	err = file.Close()
	must.NoError(ctx, err)

	liblog.Debugf("time to write cv to file %v", time.Since(t0))

	// write staging file
	// borrowBytes := x.catchupM.MarshalXdr(&cv)
	// defer borrowBytes.Recycle()

	// err = os.WriteFile(fullStaging, borrowBytes.Bytes(), 0o644)
	// must.NoError(ctx, err)

	// atomicly move staging to permanent
	err = os.Rename(fullStaging, full)
	must.NoError(ctx, err)
}

func (x *DiskStorage) FindConsensusValue(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue {
	<-x.ready

	full := x.ConsensusValueFilepath(height)

	bytes, err := os.ReadFile(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	must.NoError(ctx, err)

	cv := &proto.ConsensusValue{}
	err = libxdr.UnmarshalXdr(cv, bytes)
	must.NoError(ctx, err)

	return cv
}

func (x *DiskStorage) FindHighestConsensusValue(ctx context.Context) *proto.ConsensusValue {
	return x.FindConsensusValue(ctx, x.findConsensusValueMaxHeight(ctx))
}

func (x *DiskStorage) findConsensusValueMaxHeight(ctx context.Context) proto.BlockHeight {
	rootpath := x.ConsensusValuesDirpath()
	err := os.MkdirAll(rootpath, 0o755)
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
	must.Assertf(ctx, err == nil || errors.Is(err, errStopWalking), "walk files in reverse: %v", err)

	return maxHeight
}

var errStopWalking = fmt.Errorf("stop reverse file walk")

// scp envelopes

func (x *DiskStorage) PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.ready

	// lock ensures concurrent duplicate adds don't interfere
	// note that no lock is necessary when reclaiming or reading
	x.scpLk.Lock()
	defer x.scpLk.Unlock()

	hash := libxdr.HashXdr(x.scpM, &env)

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

func (x *DiskStorage) FindSCPEnvelopesForSlot(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope {
	<-x.ready

	found := []proto.SCPEnvelope{}

	slotpath := x.SCPEnvelopeSlotpath(slot)

	envpaths, err := ListScpFiles(slotpath)
	if err != nil {
		return nil
	}

	for _, envpath := range envpaths {
		env := proto.SCPEnvelope{}

		bytes, err := os.ReadFile(envpath)
		must.NoError(ctx, err)

		libxdr.UnmarshalXdr(&env, bytes)
		must.NoError(ctx, err)

		found = append(found, env)
	}

	return found
}
