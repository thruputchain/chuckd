package scpbroker

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/scp/scpd"
	"github.com/groundhogchain/chuckd/sub/storage"
	proto "github.com/groundhogchain/protocol-go"
)

type SCPBroker struct {
	storage    *storage.DiskStorage
	controller *scpd.SCPController
}

func NewSCPBroker(
	ctx context.Context,
	storage *storage.DiskStorage,
	controller *scpd.SCPController,
) *SCPBroker {
	return &SCPBroker{
		storage:    storage,
		controller: controller,
	}
}

// ReceivedEnvelope persists received scp envelopes, and then releases them to scp.
func (x *SCPBroker) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	x.storage.PersistSCPEnvelope(ctx, env)
	x.controller.ReceivedEnvelope(ctx, env)
}

// RepairEnvelopes retrieves past scp envelopes from persistence.
func (x *SCPBroker) RepairEnvelopes(ctx context.Context, since proto.SCPSlotIndex) []proto.SCPEnvelope {
	return x.storage.FindSCPEnvelopesSinceSlot(ctx, since)
}
