package scpbroker

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/sub/scp/scpd"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/lib-go/liblog"
	proto "github.com/groundhogchain/protocol-go"
)

const defaultBackfeedTimeout = 2 * time.Second

type SCPBackfeed interface {
	BackfeedSCPEnvelopes(ctx context.Context, since scpkit.Slot, timeout time.Duration)
	InitSCPLowestNonExternalized(ctx context.Context, slot scpkit.Slot)
}

type scpRepairer interface {
	RepairMissingSCP(ctx context.Context, slot scpkit.Slot) (proto.SCPEnvelopes, error)
}

type SCPBroker struct {
	router          router.Router
	storage         storage.SCPEnvelopePersistence
	repair          scpRepairer
	controller      *scpd.SCPController
	controllerReady chan struct{}
	stopped         atomic.Bool
	cancel          context.CancelFunc
	ctx             context.Context
}

func NewSCPBroker(
	ctx context.Context,
	router router.Router,
	storage storage.SCPEnvelopePersistence,
	repair scpRepairer,
) *SCPBroker {
	brokerCtx, cancel := context.WithCancel(ctx)
	return &SCPBroker{
		router:          router,
		storage:         storage,
		repair:          repair,
		controllerReady: make(chan struct{}),
		cancel:          cancel,
		ctx:             brokerCtx,
	}
}

func (x *SCPBroker) Stop() {
	x.stopped.CompareAndSwap(false, true)
	x.cancel()
}

func (x *SCPBroker) IsStopped() bool {
	return x.stopped.Load()
}

func (x *SCPBroker) SetSCPController(controller *scpd.SCPController) {
	x.controller = controller
	close(x.controllerReady)
}

func (x *SCPBroker) InitSCPLowestNonExternalized(ctx context.Context, slot scpkit.Slot) {
	if x.IsStopped() {
		return
	}
	x.controller.SetLowestNonExternalized(ctx, slot)
}

func (x *SCPBroker) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.controllerReady
	if x.IsStopped() {
		return
	}

	liblog.Debugf(ctx, "feeding envelope for slot %d", env.Statement.Slot_index)
	x.storage.PersistSCPEnvelope(ctx, env)
	x.controller.ReceivedEnvelope(ctx, env)
}

func (x *SCPBroker) RepairEnvelopes(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope {
	return x.storage.FindSCPEnvelopesForSlot(ctx, slot)
}

func (x *SCPBroker) BackfeedSCPEnvelopes(ctx context.Context, slot scpkit.Slot, timeout time.Duration) {
	if timeout == 0 {
		timeout = defaultBackfeedTimeout
	}
	go x.fulfillBackfeed(ctx, slot, timeout)
}

func (x *SCPBroker) fulfillBackfeed(callerCtx context.Context, slot scpkit.Slot, timeout time.Duration) {
	// Preserve caller's context values (trace IDs, log fields) but detach its cancellation.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(callerCtx), timeout)
	defer cancel()
	// Cancel when broker stops.
	stop := context.AfterFunc(x.ctx, cancel)
	defer stop()

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
		if env.Statement.Slot_index == slot {
			x.ReceivedEnvelope(ctx, env)
		}
	}
}
