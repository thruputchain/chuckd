package scpbroker

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/net/router"
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
	router          router.Router
	storage         *storage.DiskStorage
	repair          *repairagent.RepairAgent
	controller      *scpd.SCPController
	controllerReady chan struct{}
}

func NewSCPBroker(
	ctx context.Context,
	router router.Router,
	storage *storage.DiskStorage,
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

// ReceivedEnvelope persists received scp envelopes, and then releases them to scp.
func (x *SCPBroker) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.controllerReady

	x.storage.PersistSCPEnvelope(ctx, env)
	x.controller.ReceivedEnvelope(ctx, env)
}

// RepairEnvelopes retrieves past scp envelopes from persistence.
func (x *SCPBroker) RepairEnvelopes(ctx context.Context, since proto.SCPSlotIndex) []proto.SCPEnvelope {
	return x.storage.FindSCPEnvelopesForSlot(ctx, since)
}

func (x *SCPBroker) BackfeedSCPEnvelopes(ctx context.Context, slot scpkit.Slot) {
	go x.fulfillBackfeed(context.WithoutCancel(ctx), slot)
}

func (x *SCPBroker) fulfillBackfeed(ctx context.Context, slot scpkit.Slot) {

	// XXX: this needs a timeout
	envs, err := x.repair.RepairMissingSCP(ctx, slot)
	if err != nil {
		liblog.Errorf("tried to repair missing scp envelopes: %v", err)
		return
	}

	if len(envs) == 0 {
		liblog.Infof("no scp envelopes to backfeed")
	}

	for i, env := range envs {
		liblog.Debugf("backfilling envelope %d of %d", i+1, len(envs))
		// dont replay envelopes other than the requested slot
		if env.Statement.Slot_index == slot {
			x.ReceivedEnvelope(ctx, env)
		}
	}
}
