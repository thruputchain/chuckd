package baton

import (
	"context"

	"github.com/thruputchain/chuckd/sub/scp/scpkit"
)

type TestSCP struct {
	scpHandler scpkit.SCPHandler
}

func NewTestSCP(scpHandler scpkit.SCPHandler) *TestSCP {
	return &TestSCP{scpHandler: scpHandler}
}

func (x *TestSCP) CombineCandidates(ctx context.Context, slot scpkit.Slot, values []scpkit.Value) <-chan scpkit.Value {
	ch := make(chan scpkit.Value, 1)
	go func() { ch <- x.scpHandler.SCPCombineCandidates(ctx, slot, values) }()
	return ch
}

func (x *TestSCP) ValidateNominationValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) <-chan scpkit.ValidationLevel {
	ch := make(chan scpkit.ValidationLevel, 1)
	go func() { ch <- x.scpHandler.SCPValidateNominationValue(ctx, slot, value) }()
	return ch
}

func (x *TestSCP) ValidateBallotValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) <-chan scpkit.ValidationLevel {
	ch := make(chan scpkit.ValidationLevel, 1)
	go func() { ch <- x.scpHandler.SCPValidateBallotValue(ctx, slot, value) }()
	return ch
}

func (x *TestSCP) ExtractValidValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) <-chan *scpkit.Value {
	ch := make(chan *scpkit.Value, 1)
	go func() { ch <- x.scpHandler.SCPExtractValidValue(ctx, slot, value) }()
	return ch
}

func (x *TestSCP) ExternalizedValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) {
	go x.scpHandler.SCPExternalizedValue(ctx, slot, value)
}
