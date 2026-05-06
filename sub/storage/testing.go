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
	pre       map[protoutil.CVCHash]proto.CatchupBlock
	persisted map[proto.BlockHeight]proto.CatchupBlock
}

func NewTestCatchupBlockPersistence() *TestCatchupBlockPersistence {
	return &TestCatchupBlockPersistence{
		pre:       make(map[protoutil.CVCHash]proto.CatchupBlock),
		persisted: make(map[proto.BlockHeight]proto.CatchupBlock),
	}
}

func (x *TestCatchupBlockPersistence) PrePersistCatchupBlock(
	ctx context.Context,
	cb proto.CatchupBlock,
	cvch protoutil.CVCHash,
) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	x.pre[cvch] = cb
	return nil
}

func (x *TestCatchupBlockPersistence) PersistAlreadyPrePersistedCatchupBlock(
	ctx context.Context,
	height proto.BlockHeight,
	cvch protoutil.CVCHash,
) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	p, ok := x.pre[cvch]
	// dont delete p from pre, so it can be found by LookupCatchupBlock
	must.Assert(ctx, ok)
	must.Assert(ctx, p.Consensus_value.Nomination.Block_height == height)

	x.persisted[height] = p
	return nil
}

func (x *TestCatchupBlockPersistence) LookupCatchupBlock(
	ctx context.Context,
	cvch protoutil.CVCHash,
) (*proto.CatchupBlock, error) {
	x.mu.RLock()
	defer x.mu.RUnlock()

	cb, ok := x.pre[cvch]
	if !ok {
		return nil, nil
	}
	return &cb, nil
}

func (x *TestCatchupBlockPersistence) FindCatchupBlock(
	ctx context.Context,
	height proto.BlockHeight,
) (*proto.CatchupBlock, error) {
	x.mu.RLock()
	defer x.mu.RUnlock()

	cb, ok := x.persisted[height]
	if !ok {
		return nil, nil
	}
	return &cb, nil
}

type TestCVPersistence struct {
	mu          sync.RWMutex
	highest     proto.BlockHeight
	hasHighest  bool
	committed   map[proto.BlockHeight]proto.ConsensusValue
	uncommitted map[protoutil.CVCHash]proto.ConsensusValue
}

func NewTestCVPersistence() *TestCVPersistence {
	return &TestCVPersistence{
		committed:   make(map[proto.BlockHeight]proto.ConsensusValue),
		uncommitted: make(map[protoutil.CVCHash]proto.ConsensusValue),
	}
}

func (x *TestCVPersistence) PersistCV(ctx context.Context, cv proto.ConsensusValue, cvch protoutil.CVCHash) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if !x.hasHighest || cv.Nomination.Block_height > x.highest {
		x.highest = cv.Nomination.Block_height
		x.hasHighest = true
	}
	x.committed[cv.Nomination.Block_height] = cv
	x.uncommitted[cvch] = cv
	return nil
}

func (x *TestCVPersistence) PrePersistCV(ctx context.Context, cv proto.ConsensusValue, cvch protoutil.CVCHash) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.uncommitted[cvch] = cv
	return nil
}

func (x *TestCVPersistence) FindCV(ctx context.Context, height proto.BlockHeight, cvch protoutil.CVCHash) (*proto.ConsensusValue, error) {
	x.mu.RLock()
	defer x.mu.RUnlock()
	cv, ok := x.uncommitted[cvch]
	if !ok {
		return nil, nil
	}
	return &cv, nil
}

func (x *TestCVPersistence) FindCommittedCV(ctx context.Context, height proto.BlockHeight) (*proto.ConsensusValue, error) {
	x.mu.RLock()
	defer x.mu.RUnlock()
	cv, ok := x.committed[height]
	if !ok {
		return nil, nil
	}
	return &cv, nil
}

func (x *TestCVPersistence) FindHighestCommittedCV(ctx context.Context) (*proto.ConsensusValue, error) {
	x.mu.RLock()
	highest := x.highest
	hasHighest := x.hasHighest
	x.mu.RUnlock()
	if !hasHighest {
		return nil, nil
	}
	return x.FindCommittedCV(ctx, highest)
}
