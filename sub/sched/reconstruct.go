package sched

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type PartialNominationWaiter interface {
	WaitForPartialNomination(
		ctx context.Context,
		height proto.BlockHeight,
	) (proto.Nomination, proto.BlockID, error)
	UpdateNoPartialNominationsBelow(
		ctx context.Context,
		height proto.BlockHeight,
	)
}

type NominationReconstructor struct {
	lk                 sync.Mutex
	notBelow           proto.BlockHeight
	cond               *sync.Cond
	partialNominations map[proto.BlockHeight]*partialNomination
}

func NewNominationReconstructor() *NominationReconstructor {
	x := &NominationReconstructor{
		partialNominations: map[proto.BlockHeight]*partialNomination{},
	}
	x.cond = sync.NewCond(&x.lk)
	return x
}

func (x *NominationReconstructor) UpdateNoPartialNominationsBelow(ctx context.Context, height proto.BlockHeight) {

	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()

	x.notBelow = max(height, x.notBelow)
	for h := range x.partialNominations {
		if h < height {
			delete(x.partialNominations, h)
		}
	}
}

func (x *NominationReconstructor) AddChunk(ctx context.Context, chunk proto.SchedulerNominationChunk) {
	must.Assert(ctx, chunk.Total_chunks > 0)

	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()

	height := chunk.Chunk.Nomination.Block_height
	if height < x.notBelow {
		return
	}

	partial, ok := x.partialNominations[height]
	if !ok {
		partial = &partialNomination{
			blockID: chunk.Chunk.Block_id,
			total:   chunk.Total_chunks,
		}
		x.partialNominations[height] = partial
	}

	must.Assert(ctx, partial.blockID == chunk.Chunk.Block_id)
	must.Assert(ctx, partial.total == chunk.Total_chunks)

	partial.chunks = append(partial.chunks, &chunk)
}

var ErrHeightTooLow = fmt.Errorf("expected nomination height too low")

func (x *NominationReconstructor) WaitForPartialNomination(
	ctx context.Context,
	height proto.BlockHeight,
) (proto.Nomination, proto.BlockID, error) {

	x.lk.Lock()
	defer x.lk.Unlock()

	for {
		select {
		case <-ctx.Done():
			return proto.Nomination{}, 0, ctx.Err()
		default:
			if height < x.notBelow {
				return proto.Nomination{}, 0, ErrHeightTooLow
			}
			partial, ok := x.partialNominations[height]
			if ok && partial.isComplete() {
				delete(x.partialNominations, height)
				nom, blockID := partial.reconstruct()
				return nom, blockID, nil
			}
			x.cond.Wait()
		}
	}
}

type partialNomination struct {
	blockID proto.BlockID
	total   uint32
	chunks  []*proto.SchedulerNominationChunk
}

func (x *partialNomination) isComplete() bool {
	return len(x.chunks) == int(x.total)
}

func (x *partialNomination) reconstruct() (proto.Nomination, proto.BlockID) {
	sortChunks(x.chunks)
	last := len(x.chunks) - 1
	n := proto.Nomination{
		Proposer:                x.chunks[last].Chunk.Nomination.Proposer,
		Block_height:            x.chunks[last].Chunk.Nomination.Block_height,
		Block_hash:              x.chunks[last].Chunk.Nomination.Block_hash,
		Prev_block_hash:         x.chunks[last].Chunk.Nomination.Prev_block_hash,
		Transaction_hashes:      nil,
		Transaction_nominations: nil,
	}
	for _, chunk := range x.chunks {
		n.Transaction_hashes = append(n.Transaction_hashes, chunk.Chunk.Nomination.Transaction_hashes...)
		n.Transaction_nominations = append(n.Transaction_nominations, chunk.Chunk.Nomination.Transaction_nominations...)
	}
	return n, x.blockID
}

func sortChunks(chunks []*proto.SchedulerNominationChunk) {
	slices.SortFunc(chunks,
		func(x, y *proto.SchedulerNominationChunk) int {
			if x.Chunk_idx < y.Chunk_idx {
				return -1
			}
			if x.Chunk_idx == y.Chunk_idx {
				return 0
			}
			return 1
		},
	)
}
