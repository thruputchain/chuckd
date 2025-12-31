package storage

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

// XXX(perf): add cache for values read from disk
type DiskFlusher struct {
	disk *DiskStorage

	catchup *Bucket[proto.CatchupBlock, proto.BlockHeight]
	cv      *Bucket[proto.ConsensusValue, proto.BlockHeight]
	scp     *Bucket[proto.SCPEnvelope, proto.SCPSlotIndex]
}

func NewDiskFlusher(ctx context.Context, cfg Config) *DiskFlusher {
	return &DiskFlusher{
		disk:    NewDiskStorage(ctx, cfg),
		catchup: NewBucket[proto.CatchupBlock, proto.BlockHeight](),
		cv:      NewBucket[proto.ConsensusValue, proto.BlockHeight](),
		scp:     NewBucket[proto.SCPEnvelope, proto.SCPSlotIndex](),
	}
}

func (x *DiskFlusher) Start(ctx context.Context, dir string) {
	x.disk.Start(ctx, dir)
	x.catchup.Start(ctx)
	x.cv.Start(ctx)
	x.scp.Start(ctx)
}

// catchup blocks

func (x *DiskFlusher) PersistCatchupBlock(ctx context.Context, cb proto.CatchupBlock) {
	x.catchup.Add(cb)
}

func (x *DiskFlusher) FindCatchupBlock(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock {
	x.catchupFlushLk.Lock()
	defer x.catchupFlushLk.Unlock()

	unflushed := x.catchup.FindOne(height,
		func(height proto.BlockHeight, cb proto.CatchupBlock) bool {
			return cb.Consensus_value.Nomination.Block_height == height
		},
	)
	if unflushed != nil {
		return unflushed
	}
	return x.disk.FindCatchupBlock(ctx, height)
}

// cv

func (x *DiskFlusher) PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	x.cv.Add(cv)
}

func (x *DiskFlusher) FindConsensusValue(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue {
	x.cvFlushLk.Lock()
	defer x.cvFlushLk.Unlock()

	unflushed := x.cv.FindOne(height,
		func(height proto.BlockHeight, cv proto.ConsensusValue) bool {
			return cv.Nomination.Block_height == height
		},
	)
	if unflushed != nil {
		return unflushed
	}
	return x.disk.FindConsensusValue(ctx, height)
}

func (x *DiskFlusher) FindHighestConsensusValue(ctx context.Context) *proto.ConsensusValue {
	x.cvFlushLk.Lock()
	defer x.cvFlushLk.Unlock()

	unflushed := x.cv.Reduce(
		func(prev *proto.ConsensusValue, next proto.ConsensusValue) *proto.ConsensusValue {
			if prev == nil {
				return &next
			}
			if next.Nomination.Block_height > prev.Nomination.Block_height {
				return &next
			} else {
				return prev
			}
		},
	)
	flushed := x.disk.FindHighestConsensusValue(ctx)
	switch {
	case flushed == nil && unflushed == nil:
		return nil
	case flushed != nil && unflushed == nil:
		return flushed
	case flushed == nil && unflushed != nil:
		return unflushed
	default:
		if unflushed.Nomination.Block_height > flushed.Nomination.Block_height {
			return unflushed
		} else {
			return flushed
		}
	}
}

// scp envelopes

func (x *DiskFlusher) PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	x.scp.Add(env)
}

func (x *DiskFlusher) FindSCPEnvelopesForSlot(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope {
	x.scpFlushLk.Lock()
	defer x.scpFlushLk.Unlock()

	unflushed := x.scp.FindAll(slot,
		func(slot proto.SCPSlotIndex, env proto.SCPEnvelope) bool {
			return env.Statement.Slot_index == slot
		},
	)
	flushed := x.disk.FindSCPEnvelopesForSlot(ctx, slot)
	return append(flushed, unflushed...)
}
