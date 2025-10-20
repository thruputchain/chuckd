package storage

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/groundhogchain/chuckd/lib/must"
)

func (x *DiskStorage) reclaimLoop(ctx context.Context) {
	for {
		time.Sleep(x.cfg.Freq)
		x.reclaimCatchupBlocks(ctx)
		x.reclaimSCPEnvelopes(ctx)
	}
}

func (x *DiskStorage) reclaimCatchupBlocks(ctx context.Context) {

	catchupDir := x.CatchupBlocksDirpath()

	files, err := ListCatchupFiles(catchupDir)
	must.NoError(ctx, err)

	if len(files) <= x.cfg.MaxCatchupBlocks {
		return
	}

	for i := 0; i < len(files)-x.cfg.MaxCatchupBlocks; i++ {
		err = os.Remove(path.Join(catchupDir, files[i]))
		must.NoError(ctx, err)
	}
}

func (x *DiskStorage) reclaimSCPEnvelopes(ctx context.Context) {

	slotsDir := x.SCPEnvelopesDirpath()

	slotdirs, err := ListSCPSlotDirs(slotsDir)
	must.NoError(ctx, err)

	if len(slotdirs) <= x.cfg.MaxSCPSlots {
		return
	}

	for i := 0; i < len(slotdirs)-x.cfg.MaxSCPSlots; i++ {
		err = os.RemoveAll(path.Join(slotsDir, slotdirs[i]))
		must.NoError(ctx, err)
	}

}
