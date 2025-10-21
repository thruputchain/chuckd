package scpbroker

import (
	"context"

	"github.com/groundhogchain/chuckd/scp/scpd"
	"github.com/groundhogchain/chuckd/storage"
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

func (x *SCPBroker) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	x.storage.PersistSCPEnvelope(ctx, env)
	x.controller.ReceivedEnvelope(ctx, env)
}

func (x *SCPBroker) RepairEnvelopes(ctx context.Context, since proto.SCPSlotIndex) ([]proto.SCPEnvelope, error) {
	return x.storage.FindSCPEnvelopesSinceSlot(ctx, since)
}
