package scpbroker

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/repair/repairagent"
	"github.com/groundhogchain/chuckd/sub/scp/scpd"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/sub/storage"
	proto "github.com/groundhogchain/protocol-go"
)

type SCPBackfeed interface {
	BackfeedSCPEnvelopes(ctx context.Context, since scpkit.Slot)
}

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

func (x *SCPBroker) BackfeedSCPEnvelopes(ctx context.Context, since scpkit.Slot) {
	go x.fulfillBackfeed(ctx, since)
}

func (x *SCPBroker) fulfillBackfeed(ctx context.Context, since scpkit.Slot) {

	// XXX: use repair agent to fetch some envelopes

}
