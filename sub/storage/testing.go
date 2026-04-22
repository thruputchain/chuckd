package storage

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
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
	must.Assert(ctx, ok)
	must.Assert(ctx, p.Consensus_value.Nomination.Block_height == height)

	x.persisted[height] = p
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

type TestCVPersistence struct {
	mu          sync.RWMutex
	highest     proto.BlockHeight
	hasHighest  bool
	committed   map[proto.BlockHeight]proto.ConsensusValue
	uncommitted map[proto.ConsensusValueHash]proto.ConsensusValue
}

func NewTestCVPersistence() *TestCVPersistence {
	return &TestCVPersistence{
		committed:   make(map[proto.BlockHeight]proto.ConsensusValue),
		uncommitted: make(map[proto.ConsensusValueHash]proto.ConsensusValue),
	}
}

func (x *TestCVPersistence) PersistCV(ctx context.Context, cv proto.ConsensusValue, _ proto.ConsensusValueHash) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if !x.hasHighest || cv.Nomination.Block_height > x.highest {
		x.highest = cv.Nomination.Block_height
		x.hasHighest = true
	}
	x.committed[cv.Nomination.Block_height] = cv
	x.uncommitted[protoutil.HashCVFast(&cv)] = cv
}

func (x *TestCVPersistence) PrePersistCV(ctx context.Context, cv proto.ConsensusValue, _ proto.ConsensusValueHash) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.uncommitted[protoutil.HashCVFast(&cv)] = cv
}

func (x *TestCVPersistence) FindCV(ctx context.Context, height proto.BlockHeight, hash proto.ConsensusValueHash) *proto.ConsensusValue {
	x.mu.RLock()
	defer x.mu.RUnlock()
	cv, ok := x.uncommitted[hash]
	if !ok {
		return nil
	}
	return &cv
}

func (x *TestCVPersistence) FindCommittedCV(ctx context.Context, height proto.BlockHeight) *proto.ConsensusValue {
	x.mu.RLock()
	defer x.mu.RUnlock()
	cv, ok := x.committed[height]
	if !ok {
		return nil
	}
	return &cv
}

func (x *TestCVPersistence) FindHighestCommittedCV(ctx context.Context) *proto.ConsensusValue {
	x.mu.RLock()
	highest := x.highest
	hasHighest := x.hasHighest
	x.mu.RUnlock()
	if !hasHighest {
		return nil
	}
	return x.FindCommittedCV(ctx, highest)
}
