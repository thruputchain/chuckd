package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

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
	catchupPP   *PrePersist
	scpPP       *PrePersist
	consensusPP *PrePersist
	//
	scpM libxdr.XdrMarshaler
	//
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
	locked, err := flk.TryLock()
	must.NoError(ctx, err)
	must.Assertf(ctx, locked, "storage directory %v is already locked by another process", dir)

	go x.reclaimLoop(ctx)
}

// catchup blocks

func (x *DiskStorage) PrePersistCatchupBlock(
	ctx context.Context,
	cb proto.CatchupBlock,
	cvHash proto.ConsensusValueHash,
) {
	<-x.ready

	liblog.Debugf(ctx, "pre-persisting catchup block: height %d; cv_hash %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash))
	defer liblog.Debugf(ctx, "pre-persisted catchup block: height %d; cv_hash %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash))

	full := x.CatchupBlockFilepath(cb.Consensus_value.Nomination.Block_height)
	x.catchupPP.PrePersist(ctx, full, func(w io.Writer) error {
		libxdr.MarshalXdrWriter(w, &cb)
		return nil
	})
}

func (x *DiskStorage) PersistCatchupBlock(
	ctx context.Context,
	cb proto.CatchupBlock,
	cvHash proto.ConsensusValueHash,
) {
	<-x.ready

	liblog.Debugf(ctx, "persisting catchup block: height %d; cv_hash %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash))
	defer liblog.Debugf(ctx, "persisted catchup block: height %d; cv_hash %s",
		cb.Consensus_value.Nomination.Block_height, libxdr.HashString(cvHash))

	full := x.CatchupBlockFilepath(cb.Consensus_value.Nomination.Block_height)
	x.catchupPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cb)
			return nil
		},
	)
}

func (x *DiskStorage) PersistAlreadyPrePersistedCatchupBlock(
	ctx context.Context,
	height proto.BlockHeight,
	cvHash proto.ConsensusValueHash,
) {
	<-x.ready

	liblog.Debugf(ctx, "persisting already pre-persisted catchup block: height %d; cv_hash %s",
		height, libxdr.HashString(cvHash))
	defer liblog.Debugf(ctx, "persisted already pre-persisted catchup block: height %d; cv_hash %s",
		height, libxdr.HashString(cvHash))

	full := x.CatchupBlockFilepath(height)
	x.catchupPP.PersistAlreadyPrePersisted(ctx, full)
}

func (x *DiskStorage) FindCatchupBlock(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock {
	<-x.ready

	full := x.CatchupBlockFilepath(height)
	if !isPresent(full) && !isPresent(full+".prepersist") {
		return nil
	}

	cb := &proto.CatchupBlock{}
	err := x.catchupPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cb, r)
	})
	must.NoError(ctx, err)

	return cb
}

// consensus values

func (x *DiskStorage) PrePersistCV(ctx context.Context, cv proto.ConsensusValue, cvHash proto.ConsensusValueHash) {
	<-x.ready

	liblog.Debugf(ctx, "pre-persisting cv: height %d; cv_hash %s",
		cv.Nomination.Block_height, libxdr.HashString(cvHash))
	defer liblog.Debugf(ctx, "pre-persisted cv: height %d; cv_hash %s",
		cv.Nomination.Block_height, libxdr.HashString(cvHash))

	full := x.ConsensusValueFilepath(cv.Nomination.Block_height, cvHash)
	x.consensusPP.PrePersist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cv)
			return nil
		},
	)
}

func (x *DiskStorage) PersistCV(ctx context.Context, cv proto.ConsensusValue, cvHash proto.ConsensusValueHash) {
	<-x.ready

	liblog.Debugf(ctx, "persisting cv: height %d; cv_hash %s",
		cv.Nomination.Block_height, libxdr.HashString(cvHash))
	defer liblog.Debugf(ctx, "persisted cv: height %d; cv_hash %s",
		cv.Nomination.Block_height, libxdr.HashString(cvHash))

	full := x.ConsensusValueFilepath(cv.Nomination.Block_height, cvHash)
	x.consensusPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cv)
			return nil
		},
	)
}

func (x *DiskStorage) FindCV(ctx context.Context, height proto.BlockHeight, hash proto.ConsensusValueHash) *proto.ConsensusValue {
	<-x.ready

	full := x.ConsensusValueFilepath(height, hash)

	cv := &proto.ConsensusValue{}
	err := x.consensusPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cv, r)
	})
	must.NoError(ctx, err)

	return cv
}

func (x *DiskStorage) FindCommittedCV(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue {
	<-x.ready

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
	<-x.ready

	hash := libxdr.HashXdr(x.scpM, &env)
	full := x.SCPEnvelopeFilepath(env.Statement.Slot_index, hash[:5])

	x.scpPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &env)
			return nil
		},
	)
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
		err := x.scpPP.Load(ctx, envpath, func(r io.Reader) error {
			return libxdr.UnmarshalXdrReader(&env, r)
		})
		must.NoError(ctx, err)
		found = append(found, env)
	}

	return found
}
