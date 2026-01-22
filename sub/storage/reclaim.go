package storage

import (
	"context"
	"os"
	"time"

	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
)

// XXX: reclaim prepersisted cvs

func (x *DiskStorage) reclaimLoop(ctx context.Context) {
	for {
		time.Sleep(x.cfg.Freq)
		x.reclaimCatchupBlocks(ctx)
		x.reclaimSCPEnvelopes(ctx)
	}
}

func (x *DiskStorage) reclaimCatchupBlocks(ctx context.Context) {

	catchupDir := x.CatchupBlocksDirpath()

	// make sure the directory exists
	err := os.MkdirAll(catchupDir, 0o755)
	must.NoError(ctx, err)

	files, err := ListCatchupFiles(catchupDir)
	must.NoError(ctx, err)

	if len(files) <= x.cfg.MaxCatchupBlocks {
		return
	}

	for i := 0; i < len(files)-x.cfg.MaxCatchupBlocks; i++ {
		err = os.Remove(files[i])
		must.NoError(ctx, err)
	}
}

func (x *DiskStorage) reclaimSCPEnvelopes(ctx context.Context) {

	slotsDir := x.SCPEnvelopesDirpath()

	slotdirs, err := ListSCPSlotDirs(slotsDir)
	if err != nil {
		liblog.Errorf("reclaiming scp envelopes storage: %v", err)
		return
	}

	if len(slotdirs) <= x.cfg.MaxSCPSlots {
		return
	}

	for i := 0; i < len(slotdirs)-x.cfg.MaxSCPSlots; i++ {
		err = os.RemoveAll(slotdirs[i])
		must.NoError(ctx, err)
	}

}
