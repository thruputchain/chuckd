package storage

import (
	"context"
	"sync"

	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type TestCatchupBlockPersistence struct {
	mu        sync.RWMutex
	pre       map[proto.ConsensusValueHash]proto.CatchupBlock
	persisted map[proto.BlockHeight]proto.CatchupBlock
}

func NewTestCatchupBlockPersistence() *TestCatchupBlockPersistence {
	return &TestCatchupBlockPersistence{
		pre:       make(map[proto.ConsensusValueHash]proto.CatchupBlock),
		persisted: make(map[proto.BlockHeight]proto.CatchupBlock),
	}
}

func (x *TestCatchupBlockPersistence) PrePersistCatchupBlock(
	ctx context.Context,
	cb proto.CatchupBlock,
	cvHash proto.ConsensusValueHash,
) {
	x.mu.Lock()
	defer x.mu.Unlock()

	x.pre[cvHash] = cb
}

func (x *TestCatchupBlockPersistence) PersistAlreadyPrePersistedCatchupBlock(
	ctx context.Context,
	height proto.BlockHeight,
	cvHash proto.ConsensusValueHash,
) {
	x.mu.Lock()
	defer x.mu.Unlock()

	p, ok := x.pre[cvHash]
	// dont delete p from pre, so it can be found by LookupCatchupBlock
	must.Assert(ctx, ok)
	must.Assert(ctx, p.Consensus_value.Nomination.Block_height == height)

	x.persisted[height] = p
}

func (x *TestCatchupBlockPersistence) LookupCatchupBlock(
	ctx context.Context,
	cvHash proto.ConsensusValueHash,
) *proto.CatchupBlock {
	x.mu.RLock()
	defer x.mu.RUnlock()

	cb, ok := x.pre[cvHash]
	if !ok {
		return nil
	}
	return &cb
}

func (x *TestCatchupBlockPersistence) FindCatchupBlock(
	ctx context.Context,
	height proto.BlockHeight,
) *proto.CatchupBlock {
	x.mu.RLock()
	defer x.mu.RUnlock()

	cb, ok := x.persisted[height]
	if !ok {
		return nil
	}
	return &cb
}

type TestConsensusValuePersistence struct {
	mu         sync.RWMutex
	highest    proto.BlockHeight
	hasHighest bool
	height     map[proto.BlockHeight]proto.ConsensusValue
}

func NewTestConsensusValuePersistence() *TestConsensusValuePersistence {
	return &TestConsensusValuePersistence{
		height: make(map[proto.BlockHeight]proto.ConsensusValue),
	}
}

func (x *TestConsensusValuePersistence) PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue, _ proto.ConsensusValueHash) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if !x.hasHighest || cv.Nomination.Block_height > x.highest {
		x.highest = cv.Nomination.Block_height
		x.hasHighest = true
	}
	x.height[cv.Nomination.Block_height] = cv
}

func (x *TestConsensusValuePersistence) PrePersistConsensusValue(ctx context.Context, cv proto.ConsensusValue, _ proto.ConsensusValueHash) {
}

func (x *TestConsensusValuePersistence) FindConsensusValue(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue {
	x.mu.RLock()
	defer x.mu.RUnlock()
	cv, ok := x.height[height]
	if !ok {
		return nil
	}
	return &cv
}

func (x *TestConsensusValuePersistence) FindHighestConsensusValue(ctx context.Context) *proto.ConsensusValue {
	x.mu.RLock()
	highest := x.highest
	hasHighest := x.hasHighest
	x.mu.RUnlock()
	if !hasHighest {
		return nil
	}
	return x.FindConsensusValue(ctx, highest)
}
