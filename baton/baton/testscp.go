package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/scp/scpapi"
)

type TestSCP struct {
	scpHandler scpapi.SCPHandler
}

func NewTestSCP(scpHandler scpapi.SCPHandler) *TestSCP {
	return &TestSCP{scpHandler: scpHandler}
}

func (x *TestSCP) CombineCandidates(ctx context.Context, slot scpapi.Slot, values []scpapi.Value) <-chan scpapi.Value {
	ch := make(chan scpapi.Value, 1)
	go func() { ch <- x.scpHandler.CombineCandidates(ctx, slot, values) }()
	return ch
}

func (x *TestSCP) ValidateNominationValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) <-chan scpapi.ValidationLevel {
	ch := make(chan scpapi.ValidationLevel, 1)
	go func() { ch <- x.scpHandler.ValidateNominationValue(ctx, slot, value) }()
	return ch
}

func (x *TestSCP) ValidateBallotValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) <-chan scpapi.ValidationLevel {
	ch := make(chan scpapi.ValidationLevel, 1)
	go func() { ch <- x.scpHandler.ValidateBallotValue(ctx, slot, value) }()
	return ch
}

func (x *TestSCP) ExtractValidValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) <-chan *scpapi.Value {
	ch := make(chan *scpapi.Value, 1)
	go func() { ch <- x.scpHandler.ExtractValidValue(ctx, slot, value) }()
	return ch
}

func (x *TestSCP) ExternalizedValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) {
	go x.scpHandler.ExternalizedValue(ctx, slot, value)
}
