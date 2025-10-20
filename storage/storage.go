package storage

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/jay/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type DiskStorage struct {
	dir string
	cbm xdrutil.XdrMarshaler
}

func NewStorage(ctx context.Context, dir string) *DiskStorage {
	return &DiskStorage{
		dir: dir,
		cbm: xdrutil.NewXdrMarshaler(),
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
