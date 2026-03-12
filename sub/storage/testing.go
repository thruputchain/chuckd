package storage

import (
	"context"
	"sync"

	proto "github.com/groundhogchain/protocol-go"
)

type TestCatchupBlockPersistence struct {
	mu     sync.RWMutex
	blocks map[proto.BlockHeight]proto.CatchupBlock
}

func NewTestCatchupBlockPersistence() *TestCatchupBlockPersistence {
	return &TestCatchupBlockPersistence{
		blocks: make(map[proto.BlockHeight]proto.CatchupBlock),
	}
}

func (x *TestCatchupBlockPersistence) PersistCatchupBlock(ctx context.Context, cb proto.CatchupBlock) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.blocks[cb.Consensus_value.Nomination.Block_height] = cb
}

func (x *TestCatchupBlockPersistence) PrePersistCatchupBlock(ctx context.Context, cb proto.CatchupBlock) {
	x.PersistCatchupBlock(ctx, cb)
}

func (x *TestCatchupBlockPersistence) FindCatchupBlock(ctx context.Context, height proto.BlockHeight) *proto.CatchupBlock {
	x.mu.RLock()
	defer x.mu.RUnlock()
	cb, ok := x.blocks[height]
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
	hash       map[proto.ConsensusValueHash]proto.ConsensusValue
}

func NewTestConsensusValuePersistence() *TestConsensusValuePersistence {
	return &TestConsensusValuePersistence{
		height: make(map[proto.BlockHeight]proto.ConsensusValue),
		hash:   make(map[proto.ConsensusValueHash]proto.ConsensusValue),
	}
}

func (x *TestConsensusValuePersistence) PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if !x.hasHighest || cv.Nomination.Block_height > x.highest {
		x.highest = cv.Nomination.Block_height
		x.hasHighest = true
	}
	x.height[cv.Nomination.Block_height] = cv
	x.hash[cv.Nomination.Block_hash] = cv
}

func (x *TestConsensusValuePersistence) PrePersistConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	x.PersistConsensusValue(ctx, cv)
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
