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

func (x *TestSCP) CombineCandidates(ctx context.Context, slot api.Slot, values []api.Value) api.Value {
	return x.scpHandler.CombineCandidates(ctx, slot, values)
}

func (x *TestSCP) ValidateValue(ctx context.Context, slot api.Slot, value api.Value, nomination bool) api.ValidationLevel {
	return x.scpHandler.ValidateValue(ctx, slot, value, nomination)
}

func (x *TestSCP) ExtractValidValue(ctx context.Context, slot api.Slot, value api.Value) *api.Value {
	return x.scpHandler.ExtractValidValue(ctx, slot, value)
}

func (x *TestSCP) OnExternalizedValue(ctx context.Context, slot api.Slot, value api.Value) {
	x.scpHandler.OnExternalizedValue(ctx, slot, value)
}
