package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/gofrs/flock"
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
	return &DiskStorage{
		cfg:         cfg,
		catchupPP:   NewPrePersist(),
		scpPP:       NewPrePersist(),
		consensusPP: NewPrePersist(),
		scpM:        libxdr.NewXdrMarshaler(),
		ready:       make(chan struct{}),
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

	full := x.CatchupBlockFilepath(cb.Consensus_value.Nomination.Block_height)
	x.catchupPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cb)
			return nil
		},
	)
}

func (x *DiskStorage) FindCatchupBlock(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock {
	<-x.ready

	full := x.CatchupBlockFilepath(height)
	if !isPresent(full) {
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

func (x *DiskStorage) PrePersistConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	<-x.ready

	full := x.ConsensusValueFilepath(cv.Nomination.Block_height)
	x.consensusPP.PrePersist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cv)
			return nil
		},
	)
}

func (x *DiskStorage) PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	<-x.ready

	full := x.ConsensusValueFilepath(cv.Nomination.Block_height)
	x.consensusPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cv)
			return nil
		},
	)
}

func (x *DiskStorage) FindConsensusValue(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue {
	<-x.ready

	full := x.ConsensusValueFilepath(height)
	if !isPresent(full) {
		return nil
	}

	cv := &proto.ConsensusValue{}
	err := x.consensusPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cv, r)
	})
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

	hash := libxdr.HashXdr(x.scpM, &env)
	full := x.SCPEnvelopeFilepath(env.Statement.Slot_index, time.Now(), hash[:5])

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
		err := x.consensusPP.Load(ctx, envpath, func(r io.Reader) error {
			return libxdr.UnmarshalXdrReader(&env, r)
		})
		must.NoError(ctx, err)
		found = append(found, env)
	}

	return found
}
