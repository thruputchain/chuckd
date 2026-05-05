package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"runtime"
	"sync/atomic"

	"github.com/gofrs/flock"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type DiskStorage struct {
	dir string
	cfg Config
	//
	catchupPP   *PrePersist
	scpPP       *PrePersist
	consensusPP *PrePersist
	//
	scpM libxdr.XdrMarshaler
	//
	flk   atomic.Pointer[flock.Flock]
	ready chan struct{}
}

func NewStorage(ctx context.Context, cfg Config) *DiskStorage {
	s := &DiskStorage{
		cfg:         cfg,
		catchupPP:   NewPrePersist(),
		scpPP:       NewPrePersist(),
		consensusPP: NewPrePersist(),
		scpM:        libxdr.NewXdrMarshaler(),
		ready:       make(chan struct{}),
	}
	var _ CatchupBlockPersistence = s
	var _ ConsensusValuePersistence = s
	var _ SCPEnvelopePersistence = s
	return s
}

func (x *DiskStorage) Start(ctx context.Context, dir string) {
	defer close(x.ready)

	x.dir = dir

	// obtain an advisory lock
	err := os.MkdirAll(dir, 0o755)
	must.NoError(ctx, err)

	flk := flock.New(path.Join(dir, "lock"))
	x.flk.Store(flk)
	locked, err := flk.TryLock()
	must.NoError(ctx, err)
	must.Assertf(ctx, locked, "storage directory %v is already locked by another process", dir)

	go x.reclaimLoop(ctx)
}

func (x *DiskStorage) Stop() {
	flk := x.flk.Swap(nil)
	if flk != nil {
		flk.Unlock()
	}
	runtime.GC()
}

// catchup blocks

// WaitReady blocks until the storage is ready to be used and returns nil.
// It returns non-nil error, if the storage system has been stopped or the context is done.
func (x *DiskStorage) WaitReady(ctx context.Context) {
	select {
	case <-ctx.Done():
		must.Assertf(ctx, false, "storage operation after context done")
	case <-x.ready:
		flk := x.flk.Load()
		must.Assertf(ctx, flk != nil, "storage operation after stop")
	}
}

func (x *DiskStorage) PrePersistCatchupBlock(
	ctx context.Context,
	cb proto.CatchupBlock,
	cvch protoutil.CVCHash,
) {
	x.WaitReady(ctx)

	liblog.Debugf(ctx, "pre-persisting catchup block: height %d; cvc %s",
		cb.Consensus_value.Nomination.Block_height,
		libxdr.HashString(cvch))
	defer liblog.Debugf(ctx, "pre-persisted catchup block: height %d; cvc %s",
		cb.Consensus_value.Nomination.Block_height,
		libxdr.HashString(cvch))

	full := x.CatchupBlockFilepath(cb.Consensus_value.Nomination.Block_height, cvch)
	x.catchupPP.PrePersist(ctx, full, func(w io.Writer) error {
		libxdr.MarshalXdrWriter(w, &cb)
		return nil
	})
}

func (x *DiskStorage) PersistCatchupBlock(
	ctx context.Context,
	cb proto.CatchupBlock,
	cvch protoutil.CVCHash,
) {
	x.WaitReady(ctx)

	liblog.Debugf(ctx, "persisting catchup block: height %d; cvc %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvch))
	defer liblog.Debugf(ctx, "persisted catchup block: height %d; cvc %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvch))

	full := x.CatchupBlockFilepath(cb.Consensus_value.Nomination.Block_height, cvch)
	x.catchupPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cb)
			return nil
		},
	)
}

// LookupCatchupBlock finds persisted or pre-persisted catchup blocks with given cv hash.
// Blocks we have nominated but are not committed will be found in pre-persist.
func (x *DiskStorage) LookupCatchupBlock(
	ctx context.Context,
	cvch protoutil.CVCHash,
) *proto.CatchupBlock {
	x.WaitReady(ctx)

	full, err := FindCatchupFileByHash(x.CatchupBlocksDirpath(), cvch)
	must.NoError(ctx, err)
	if full == "" {
		return nil
	}

	cb := &proto.CatchupBlock{}
	err = x.catchupPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cb, r)
	})
	must.NoError(ctx, err)

	return cb
}

func (x *DiskStorage) PersistAlreadyPrePersistedCatchupBlock(
	ctx context.Context,
	height proto.BlockHeight,
	cvch protoutil.CVCHash,
) {
	x.WaitReady(ctx)

	liblog.Debugf(ctx, "persisting already pre-persisted catchup block: height %d; cvc %s",
		height, libxdr.HashString(cvch))
	defer liblog.Debugf(ctx, "persisted already pre-persisted catchup block: height %d; cvc %s",
		height, libxdr.HashString(cvch))

	full := x.CatchupBlockFilepath(height, cvch)
	x.catchupPP.PersistAlreadyPrePersisted(ctx, full)
}

func (x *DiskStorage) FindCatchupBlock(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock {
	x.WaitReady(ctx)

	files, err := ListCatchupFilesAtHeight(x.CatchupBlocksDirpath(), height)
	must.NoError(ctx, err)
	if len(files) == 0 {
		return nil
	}
	must.Assertf(ctx, len(files) == 1, "expected exactly one catchup file at height %d, got %d", height, len(files))
	full := files[len(files)-1]

	cb := &proto.CatchupBlock{}
	err = x.catchupPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cb, r)
	})
	must.NoError(ctx, err)

	return cb
}

// consensus values

func (x *DiskStorage) PersistCV(ctx context.Context, cv proto.ConsensusValue, cvch protoutil.CVCHash) {
	x.WaitReady(ctx)

	liblog.Debugf(ctx, "persisting cv: height %d; cvc %s",
		cv.Nomination.Block_height, libxdr.HashString(cvch))
	defer liblog.Debugf(ctx, "persisted cv: height %d; cv_hash %s",
		cv.Nomination.Block_height, libxdr.HashString(cvch))

	full := x.ConsensusValueFilepath(cv.Nomination.Block_height, cvch)
	x.consensusPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cv)
			return nil
		},
	)
}

func (x *DiskStorage) PrePersistCV(ctx context.Context, cv proto.ConsensusValue, cvch protoutil.CVCHash) {
	x.WaitReady(ctx)

	liblog.Debugf(ctx, "pre-persisting cv: height %d; cvc %s",
		cv.Nomination.Block_height, libxdr.HashString(cvch))
	defer liblog.Debugf(ctx, "pre-persisted cv: height %d; cvc %s",
		cv.Nomination.Block_height, libxdr.HashString(cvch))

	full := x.ConsensusValueFilepath(cv.Nomination.Block_height, cvch)
	x.consensusPP.PrePersist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cv)
			return nil
		},
	)
}

func (x *DiskStorage) FindCV(ctx context.Context, height proto.BlockHeight, cvch protoutil.CVCHash) *proto.ConsensusValue {
	x.WaitReady(ctx)

	full := x.ConsensusValueFilepath(height, cvch)

	cv := &proto.ConsensusValue{}
	err := x.consensusPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cv, r)
	})
	must.NoError(ctx, err)

	return cv
}

func (x *DiskStorage) FindCommittedCV(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue {
	x.WaitReady(ctx)

	files, err := x.ListConsensusValueFilesAtHeight(height)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		must.NoError(ctx, err)
	}
	if len(files) == 0 {
		return nil
	}
	must.Assertf(ctx, len(files) == 1, "expected exactly one committed consensus value file at height %d, got %d", height, len(files))
	full := files[len(files)-1]

	cv := &proto.ConsensusValue{}
	err = x.consensusPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cv, r)
	})
	must.NoError(ctx, err)

	return cv
}

func (x *DiskStorage) FindHighestCommittedCV(ctx context.Context) *proto.ConsensusValue {
	return x.FindCommittedCV(ctx, x.findConsensusValueMaxHeight(ctx))
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
	x.WaitReady(ctx)

	hash := libxdr.HashXdr(x.scpM, &env)
	full := x.SCPEnvelopeFilepath(env.Statement.Slot_index, hash[:5])

	x.scpPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &env)
			return nil
		},
	)
}

func (x *DiskStorage) PersistSCPEnvelopeFromLocal(ctx context.Context, env proto.SCPEnvelope) {
	x.WaitReady(ctx)

	full := x.SCPEnvelopeFromLocalFilepath(env.Statement.Slot_index)

	x.scpPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &env)
			return nil
		},
	)
}

func (x *DiskStorage) FindSCPEnvelopesForSlotFromLocal(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope {
	x.WaitReady(ctx)

	full := x.SCPEnvelopeFromLocalFilepath(slot)

	if !isPresent(full) {
		return nil
	}

	env := proto.SCPEnvelope{}
	err := x.scpPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(&env, r)
	})
	must.NoError(ctx, err)

	return []proto.SCPEnvelope{env}
}

func (x *DiskStorage) FindSCPEnvelopesForSlot(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope {
	x.WaitReady(ctx)

	found := []proto.SCPEnvelope{}

	slotpath := x.SCPEnvelopeSlotpath(slot)

	envpaths, err := ListScpFiles(slotpath)
	if err != nil {
		return nil
	}

	for _, envpath := range envpaths {
		env := proto.SCPEnvelope{}
		err := x.scpPP.Load(ctx, envpath, func(r io.Reader) error {
			return libxdr.UnmarshalXdrReader(&env, r)
		})
		must.NoError(ctx, err)
		found = append(found, env)
	}

	return found
}

// ListSlotsWithSCPEnvelopesNotBelow lists all slots not below a given slot that have persisted scp envelopes.
// The slots are returned in ascending order.
func (x *DiskStorage) ListSlotsWithSCPEnvelopesNotBelow(ctx context.Context, below proto.SCPSlotIndex) []proto.SCPSlotIndex {
	x.WaitReady(ctx)

	_, allFound, err := ListSCPSlotDirs(x.SCPEnvelopesDirpath())
	must.NoError(ctx, err)

	found := []proto.SCPSlotIndex{}
	for _, index := range allFound {
		if index < below {
			continue
		}
		found = append(found, index)
	}

	return found
}
