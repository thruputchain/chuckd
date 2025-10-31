package scpbroker

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/repair/repairagent"
	"github.com/groundhogchain/chuckd/sub/scp/scpd"
	"github.com/groundhogchain/chuckd/sub/storage"
	proto "github.com/groundhogchain/protocol-go"
)

type SCPBroker struct {
	storage         *storage.DiskStorage
	repair          *repairagent.RepairAgent
	controller      *scpd.SCPController
	controllerReady chan struct{}
}

func NewSCPBroker(
	ctx context.Context,
	storage *storage.DiskStorage,
	repair *repairagent.RepairAgent,
) *SCPBroker {
	return &SCPBroker{
		storage:         storage,
		repair:          repair,
		controllerReady: make(chan struct{}),
	}
}

func (x *SCPBroker) SetSCPController(controller *scpd.SCPController) {
	x.controller = controller
	close(x.controllerReady)
}

// ReceivedEnvelope persists received scp envelopes, and then releases them to scp.
func (x *SCPBroker) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.controllerReady
	x.storage.PersistSCPEnvelope(ctx, env)
	x.controller.ReceivedEnvelope(ctx, env)
}

// RepairEnvelopes retrieves past scp envelopes from persistence.
func (x *SCPBroker) RepairEnvelopes(ctx context.Context, since proto.SCPSlotIndex) []proto.SCPEnvelope {
	return x.storage.FindSCPEnvelopesSinceSlot(ctx, since)
}
