package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/scp/api"
)

type TestSCP struct {
	scpHandler api.SCPHandler
}

func NewTestSCP(scpHandler api.SCPHandler) *TestSCP {
	return &TestSCP{scpHandler: scpHandler}
}

func (x *TestSCP) CombineCandidates(ctx context.Context, slot api.Slot, values []api.Value) <-chan api.Value {
	ch := make(chan api.Value, 1)
	go func() { ch <- x.scpHandler.CombineCandidates(ctx, slot, values) }()
	return ch
}

func (x *TestSCP) ValidateValue(ctx context.Context, slot api.Slot, value api.Value, nomination bool) <-chan api.ValidationLevel {
	ch := make(chan api.ValidationLevel, 1)
	go func() { ch <- x.scpHandler.ValidateValue(ctx, slot, value, nomination) }()
	return ch
}

func (x *TestSCP) ExtractValidValue(ctx context.Context, slot api.Slot, value api.Value) <-chan *api.Value {
	ch := make(chan *api.Value, 1)
	go func() { ch <- x.scpHandler.ExtractValidValue(ctx, slot, value) }()
	return ch
}

func (x *TestSCP) ExternalizedValue(ctx context.Context, slot api.Slot, value api.Value) {
	go x.scpHandler.ExternalizedValue(ctx, slot, value)
}
