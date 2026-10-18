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
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
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
func (x *DiskStorage) WaitReady(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-x.ready:
		flk := x.flk.Load()
		must.Assertf(ctx, flk != nil, "storage operation after stop")
		return nil
	}
}

func (x *DiskStorage) PrePersistCatchupBlock(
	ctx context.Context,
	cb proto.CatchupBlock,
	vhash protoutil.ValueHash,
) error {
	if err := x.WaitReady(ctx); err != nil {
		return err
	}

	liblog.Debugf(ctx, "pre-persisting catchup block: height %d; value %s",
		cb.Consensus_basis.Value.Proposal_header.Block_height, vhash.Display())
	defer liblog.Debugf(ctx, "pre-persisted catchup block: height %d; value %s",
		cb.Consensus_basis.Value.Proposal_header.Block_height, vhash.Display())

	full := x.CatchupBlockFilepath(cb.Consensus_basis.Value.Proposal_header.Block_height, vhash)
	x.catchupPP.PrePersist(ctx, full, func(w io.Writer) error {
		libxdr.MarshalXdrWriter(w, &cb)
		return nil
	})

	return nil
}

func (x *DiskStorage) PersistCatchupBlock(
	ctx context.Context,
	cb proto.CatchupBlock,
	vhash protoutil.ValueHash,
) error {
	if err := x.WaitReady(ctx); err != nil {
		return err
	}

	liblog.Debugf(ctx, "persisting catchup block: height %d; value %s",
		cb.Consensus_basis.Value.Proposal_header.Block_height, vhash.Display())
	defer liblog.Debugf(ctx, "persisted catchup block: height %d; value %s",
		cb.Consensus_basis.Value.Proposal_header.Block_height, vhash.Display())

	full := x.CatchupBlockFilepath(cb.Consensus_basis.Value.Proposal_header.Block_height, vhash)
	x.catchupPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &cb)
			return nil
		},
	)

	return nil
}

// LookupCatchupBlock finds persisted or pre-persisted catchup blocks with given cv hash.
// Blocks we have nominated but are not committed will be found in pre-persist.
func (x *DiskStorage) LookupCatchupBlock(
	ctx context.Context,
	vhash protoutil.ValueHash,
) (*proto.CatchupBlock, error) {
	if err := x.WaitReady(ctx); err != nil {
		return nil, err
	}

	full, err := FindCatchupFileByHash(x.CatchupBlocksDirpath(), vhash)
	must.NoError(ctx, err)
	if full == "" {
		return nil, nil
	}

	cb := &proto.CatchupBlock{}
	err = x.catchupPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cb, r)
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	must.NoError(ctx, err)

	return cb, nil
}

func (x *DiskStorage) PersistAlreadyPrePersistedCatchupBlock(
	ctx context.Context,
	height proto.BlockHeight,
	vhash protoutil.ValueHash,
) error {
	if err := x.WaitReady(ctx); err != nil {
		return err
	}

	liblog.Debugf(ctx, "persisting already pre-persisted catchup block: height %d; value %s",
		height, vhash.Display())
	defer liblog.Debugf(ctx, "persisted already pre-persisted catchup block: height %d; value %s",
		height, vhash.Display())

	full := x.CatchupBlockFilepath(height, vhash)
	x.catchupPP.PersistAlreadyPrePersisted(ctx, full)

	return nil
}

func (x *DiskStorage) FindCatchupBlock(ctx context.Context, height proto.BlockHeight) (*proto.CatchupBlock, error) {
	if err := x.WaitReady(ctx); err != nil {
		return nil, err
	}

	files, err := ListCatchupFilesAtHeight(x.CatchupBlocksDirpath(), height)
	must.NoError(ctx, err)
	if len(files) == 0 {
		return nil, nil
	}
	must.Assertf(ctx, len(files) == 1, "expected exactly one catchup file at height %d, got %d", height, len(files))
	full := files[len(files)-1]

	cb := &proto.CatchupBlock{}
	err = x.catchupPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(cb, r)
	})
	// os.ErrNotExist can only occur if the disk is corrupted.
	must.NoError(ctx, err)

	return cb, nil
}

// consensus values

func (x *DiskStorage) PersistCV(ctx context.Context, basis proto.ConsensusBasis, vhash protoutil.ValueHash) error {
	if err := x.WaitReady(ctx); err != nil {
		return err
	}

	liblog.Debugf(ctx, "persisting cv: height %d; value %s",
		protoutil.ValueHeight(&basis.Value), vhash.Display())
	defer liblog.Debugf(ctx, "persisted cv: height %d; value %s",
		protoutil.ValueHeight(&basis.Value), vhash.Display())

	full := x.ConsensusValueFilepath(protoutil.ValueHeight(&basis.Value), vhash)
	x.consensusPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &basis)
			return nil
		},
	)

	return nil
}

func (x *DiskStorage) PrePersistCV(ctx context.Context, basis proto.ConsensusBasis, vhash protoutil.ValueHash) error {
	if err := x.WaitReady(ctx); err != nil {
		return err
	}

	liblog.Debugf(ctx, "pre-persisting cv: height %d; value %s",
		protoutil.ValueHeight(&basis.Value), vhash.Display())
	defer liblog.Debugf(ctx, "pre-persisted cv: height %d; value %s",
		protoutil.ValueHeight(&basis.Value), vhash.Display())

	full := x.ConsensusValueFilepath(protoutil.ValueHeight(&basis.Value), vhash)
	x.consensusPP.PrePersist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &basis)
			return nil
		},
	)

	return nil
}

func (x *DiskStorage) FindCV(ctx context.Context, height proto.BlockHeight, vhash protoutil.ValueHash) (*proto.ConsensusBasis, error) {
	if err := x.WaitReady(ctx); err != nil {
		return nil, err
	}

	full := x.ConsensusValueFilepath(height, vhash)

	basis := &proto.ConsensusBasis{}
	err := x.consensusPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(basis, r)
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	must.NoError(ctx, err)

	return basis, nil
}

func (x *DiskStorage) FindCommittedCV(ctx context.Context, height proto.BlockHeight) (*proto.ConsensusBasis, error) {
	if err := x.WaitReady(ctx); err != nil {
		return nil, err
	}

	files, err := x.ListConsensusValueFilesAtHeight(height)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		must.NoError(ctx, err)
	}
	if len(files) == 0 {
		return nil, nil
	}
	must.Assertf(ctx, len(files) == 1, "expected exactly one committed consensus value file at height %d, got %d", height, len(files))
	full := files[len(files)-1]

	basis := &proto.ConsensusBasis{}
	err = x.consensusPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(basis, r)
	})
	// os.ErrNotExist can only occur if the disk is corrupted.
	must.NoError(ctx, err)

	return basis, nil
}

func (x *DiskStorage) FindHighestCommittedCV(ctx context.Context) (*proto.ConsensusBasis, error) {
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

func (x *DiskStorage) PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) error {
	if err := x.WaitReady(ctx); err != nil {
		return err
	}

	hash := libxdr.HashXdr(x.scpM, &env)
	full := x.SCPEnvelopeFilepath(env.Statement.Slot_index, hash[:])

	x.scpPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &env)
			return nil
		},
	)

	return nil
}

func (x *DiskStorage) PersistSCPEnvelopeFromLocal(ctx context.Context, env proto.SCPEnvelope) error {
	if err := x.WaitReady(ctx); err != nil {
		return err
	}

	full := x.SCPEnvelopeFromLocalFilepath(env.Statement.Slot_index)

	x.scpPP.Persist(ctx, full,
		func(w io.Writer) error {
			libxdr.MarshalXdrWriter(w, &env)
			return nil
		},
	)

	return nil
}

func (x *DiskStorage) FindSCPEnvelopesForSlotFromLocal(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error) {
	if err := x.WaitReady(ctx); err != nil {
		return nil, err
	}

	full := x.SCPEnvelopeFromLocalFilepath(slot)

	if !isPresent(full) {
		return nil, nil
	}

	env := proto.SCPEnvelope{}
	err := x.scpPP.Load(ctx, full, func(r io.Reader) error {
		return libxdr.UnmarshalXdrReader(&env, r)
	})
	// os.ErrNotExist can only occur if the disk is corrupted.
	must.NoError(ctx, err)

	return []proto.SCPEnvelope{env}, nil
}

func (x *DiskStorage) FindSCPEnvelopesForSlot(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error) {
	if err := x.WaitReady(ctx); err != nil {
		return nil, err
	}

	found := []proto.SCPEnvelope{}

	slotpath := x.SCPEnvelopeSlotpath(slot)

	envpaths, err := ListScpFiles(slotpath)
	if err != nil {
		return nil, nil
	}

	for _, envpath := range envpaths {
		env := proto.SCPEnvelope{}
		err := x.scpPP.Load(ctx, envpath, func(r io.Reader) error {
			return libxdr.UnmarshalXdrReader(&env, r)
		})
		// os.ErrNotExist can only occur if the disk is corrupted.
		must.NoError(ctx, err)
		found = append(found, env)
	}

	return found, nil
}

// ListSlotsWithSCPEnvelopesNotBelow lists all slots not below a given slot that have persisted scp envelopes.
// The slots are returned in ascending order.
func (x *DiskStorage) ListSlotsWithSCPEnvelopesNotBelow(ctx context.Context, below proto.SCPSlotIndex) ([]proto.SCPSlotIndex, error) {
	if err := x.WaitReady(ctx); err != nil {
		return nil, err
	}

	_, allFound, err := ListSCPSlotDirs(x.SCPEnvelopesDirpath())
	must.NoError(ctx, err)

	found := []proto.SCPSlotIndex{}
	for _, index := range allFound {
		if index < below {
			continue
		}
		found = append(found, index)
	}

	return found, nil
}
