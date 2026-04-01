package scpbroker

import (
	"context"
	"sync"

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
	SetLowestNonExternalized(ctx context.Context, slot scpkit.Slot)
}

type SCPBroker struct {
	router                router.Router
	storage               storage.SCPEnvelopePersistence
	repair                *repairagent.RepairAgent
	controller            *scpd.SCPController
	controllerReady       chan struct{}
	lk                    sync.Mutex
	lowestNonExternalized scpkit.Slot
}

func NewSCPBroker(
	ctx context.Context,
	router router.Router,
	storage storage.SCPEnvelopePersistence,
	repair *repairagent.RepairAgent,
) *SCPBroker {
	return &SCPBroker{
		router:                router,
		storage:               storage,
		repair:                repair,
		controllerReady:       make(chan struct{}),
		lowestNonExternalized: 0,
	}
}

func (x *SCPBroker) SetSCPController(controller *scpd.SCPController) {
	x.controller = controller
	close(x.controllerReady)
}

func (x *SCPBroker) SetLowestNonExternalized(ctx context.Context, slot scpkit.Slot) {
	// <-x.controllerReady
	x.controller.SetLowestNonExternalized(ctx, slot)

	x.lk.Lock()
	defer x.lk.Unlock()
	x.lowestNonExternalized = max(x.lowestNonExternalized, slot)
}

func (x *SCPBroker) GetLowestNonExternalized(ctx context.Context) scpkit.Slot {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.lowestNonExternalized
}

// ReceivedEnvelope persists received scp envelopes, and then releases them to scp.
func (x *SCPBroker) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.controllerReady

	// Prevent envelopes below the lowest non-externalized slot from being sent to scpd.
	// This is not a complete solution, because scpd may have already moved to a higher externalized slot.
	// Scpd itself also prevents envelopes below its highest externalized slot from being processed.
	lowestNonExternalized := x.GetLowestNonExternalized(ctx)
	if env.Statement.Slot_index < lowestNonExternalized {
		liblog.Infof("skipping envelope for slot %d which is below the lowest non-externalized slot %d",
			env.Statement.Slot_index,
			lowestNonExternalized)
		return
	}

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
