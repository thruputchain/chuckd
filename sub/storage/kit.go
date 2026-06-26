package storage

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: interaction between storage and protocol upgrades?

type CatchupBlockPersistence interface {
	PrePersistCatchupBlock(
		ctx context.Context,
		cb proto.CatchupBlock,
		cvch protoutil.CVCHash,
	) error

	PersistAlreadyPrePersistedCatchupBlock(
		ctx context.Context,
		height proto.BlockHeight,
		cvch protoutil.CVCHash,
	) error

	LookupCatchupBlock(
		ctx context.Context,
		cvch protoutil.CVCHash,
	) (*proto.CatchupBlock, error)

	FindCatchupBlock(
		ctx context.Context,
		height proto.BlockHeight,
	) (*proto.CatchupBlock, error)
}

type ConsensusValuePersistence interface {
	PersistCV(ctx context.Context, basis proto.ConsensusBasis, hash protoutil.CVCHash) error
	PrePersistCV(ctx context.Context, basis proto.ConsensusBasis, hash protoutil.CVCHash) error
	FindCV(ctx context.Context, height proto.BlockHeight, hash protoutil.CVCHash) (*proto.ConsensusBasis, error)
	FindCommittedCV(ctx context.Context, height proto.BlockHeight) (*proto.ConsensusBasis, error)
	FindHighestCommittedCV(ctx context.Context) (*proto.ConsensusBasis, error)
}

func FindCommittedConsensusValuesAbove(ctx context.Context, x ConsensusValuePersistence, above proto.BlockHeight) ([]proto.ConsensusBasis, error) {
	found := []proto.ConsensusBasis{}
	highest, err := x.FindHighestCommittedCV(ctx)
	if err != nil {
		return nil, err
	}
	if highest == nil {
		return nil, nil
	}
	for height := above + 1; height <= highest.Nomination.Block_height; height++ {
		basis, err := x.FindCommittedCV(ctx, height)
		if err != nil {
			return nil, err
		}
		if basis == nil {
			continue
		}
		found = append(found, *basis)
	}
	return found, nil
}

type SCPEnvelopePersistence interface {
	PersistSCPEnvelope(ctx context.Context, env proto.SCPEnvelope) error

	PersistSCPEnvelopeFromLocal(ctx context.Context, env proto.SCPEnvelope) error

	FindSCPEnvelopesForSlot(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error)

	FindSCPEnvelopesForSlotFromLocal(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error)

	// ListSlotsWithSCPEnvelopesNotBelow lists all slots not below a given slot that have persisted scp envelopes.
	// The slots are returned in ascending order.
	ListSlotsWithSCPEnvelopesNotBelow(ctx context.Context, below proto.SCPSlotIndex) ([]proto.SCPSlotIndex, error)
}

type Config struct {
	Freq             time.Duration // garbage collection loop freq
	MaxCatchupBlocks int           // catchup blocks
	MaxSCPSlots      int           // scp envelopes
}
