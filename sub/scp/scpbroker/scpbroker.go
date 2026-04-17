package scpbroker

import (
	"context"

	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/sub/repair/repairagent"
	"github.com/groundhogchain/chuckd/sub/scp/scpd"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/lib-go/liblog"
	proto "github.com/groundhogchain/protocol-go"
)

type SCPBackfeed interface {
	BackfeedSCPEnvelopes(ctx context.Context, since scpkit.Slot)
	InitSCPLowestNonExternalized(ctx context.Context, slot scpkit.Slot)
}

type SCPBroker struct {
	router          router.Router
	storage         storage.SCPEnvelopePersistence
	repair          *repairagent.RepairAgent
	controller      *scpd.SCPController
	controllerReady chan struct{}
}

func NewSCPBroker(
	ctx context.Context,
	router router.Router,
	storage storage.SCPEnvelopePersistence,
	repair *repairagent.RepairAgent,
) *SCPBroker {
	return &SCPBroker{
		router:          router,
		storage:         storage,
		repair:          repair,
		controllerReady: make(chan struct{}),
	}
}

func (x *SCPBroker) SetSCPController(controller *scpd.SCPController) {
	x.controller = controller
	close(x.controllerReady)
}

func (x *SCPBroker) InitSCPLowestNonExternalized(ctx context.Context, slot scpkit.Slot) {
	x.controller.SetLowestNonExternalized(ctx, slot)
}

// ReceivedEnvelope persists received scp envelopes, and then releases them to scp.
func (x *SCPBroker) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.controllerReady

	x.storage.PersistSCPEnvelope(ctx, env)
	x.controller.ReceivedEnvelope(ctx, env)
}

// RepairEnvelopes retrieves past scp envelopes from persistence.
func (x *SCPBroker) RepairEnvelopes(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope {
	return x.storage.FindSCPEnvelopesForSlot(ctx, slot)
}

func (x *SCPBroker) BackfeedSCPEnvelopes(ctx context.Context, slot scpkit.Slot) {
	go x.fulfillBackfeed(context.WithoutCancel(ctx), slot)
}

func (x *SCPBroker) fulfillBackfeed(ctx context.Context, slot scpkit.Slot) {

	// XXX: this needs a timeout
	envs, err := x.repair.RepairMissingSCP(ctx, slot)
	if err != nil {
		liblog.Errorf(ctx, "tried to repair missing scp envelopes for slot %d: %v", slot, err)
		return
	}

	if len(envs) == 0 {
		liblog.Infof(ctx, "no scp envelopes to backfeed")
	}

	for i, env := range envs {
		liblog.Debugf(ctx, "backfilling envelope %d of %d for slot %d", i+1, len(envs), env.Statement.Slot_index)
		// dont replay envelopes other than the requested slot
		if env.Statement.Slot_index == slot {
			x.ReceivedEnvelope(ctx, env)
		}
	}
}
