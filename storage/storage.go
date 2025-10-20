package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/jay/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type DiskStorage struct {
	dir  string
	cbm  xdrutil.XdrMarshaler
	scpm xdrutil.XdrMarshaler
}

func NewStorage(ctx context.Context, dir string) *DiskStorage {
	//XXX: advisory lock directory
	return &DiskStorage{
		dir:  dir,
		cbm:  xdrutil.NewXdrMarshaler(),
		scpm: xdrutil.NewXdrMarshaler(),
	}
}

// catchup blocks

func (x *DiskStorage) AddCatchupBlock(ctx context.Context, cb proto.CatchupBlock) {

	full := x.CatchupBlockFilepath(cb.Consensus_value.Nomination.Block_height)
	fullStaging := path.Join(filepath.Dir(full), filepath.Base(full)+".staging-"+FilenameNonce(ctx))

	err := os.MkdirAll(filepath.Dir(full), 0o755)
	must.NoError(ctx, err)

	// write staging file
	borrowBytes := x.cbm.MarshalXdr(&cb)
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

	stamp := protoutil.TimeFromTimepoint(cv.Timestamp)
	full := x.ConsensusValueFilepath(stamp, cv.Nomination.Block_height)
	fullStaging := path.Join(filepath.Dir(full), filepath.Base(full)+".staging-"+FilenameNonce(ctx))

	err := os.MkdirAll(filepath.Dir(full), 0o755)
	must.NoError(ctx, err)

	// write staging file
	borrowBytes := x.cbm.MarshalXdr(&cv)
	defer borrowBytes.Recycle()

	err = os.WriteFile(fullStaging, borrowBytes.Bytes(), 0o644)
	must.NoError(ctx, err)

	// atomicly move staging to permanent
	err = os.Rename(fullStaging, full)
	must.NoError(ctx, err)
}

var errStopWalking = fmt.Errorf("stop reverse file walk")

func (x *DiskStorage) FindLatestConsensusValues(ctx context.Context, n int) ([]proto.ConsensusValue, error) {

	rootpath := x.ConsensusValueRootpath()
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

	hash := xdrutil.HashXdr(x.scpm, &env)

	full := x.SCPEnvelopeFilepath(env.Statement.Slot_index, time.Now(), hash[:5])
	fullStaging := path.Join(filepath.Dir(full), filepath.Base(full)+".staging-"+FilenameNonce(ctx))

	err := os.MkdirAll(filepath.Dir(full), 0o755)
	must.NoError(ctx, err)

	// write staging file
	borrowBytes := x.scpm.MarshalXdr(&env)
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

			bytes, err := os.ReadFile(path.Join(slotpath, envpath))
			must.NoError(ctx, err)

			xdrutil.UnmarshalXdr(&env, bytes)
			must.NoError(ctx, err)

			found = append(found, env)
		}
	}

	return found, nil
}
