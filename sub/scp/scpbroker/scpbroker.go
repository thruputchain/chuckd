package scpbroker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/sub/scp/scpd"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/lib-go/liblog"
	proto "github.com/groundhogchain/protocol-go"
)

type SCPRepairer interface {
	RepairMissingSCP(ctx context.Context, slot scpkit.Slot) (proto.SCPEnvelopes, error)
}

type SCPBroker struct {
	router          router.Router
	storage         storage.SCPEnvelopePersistence
	repair          SCPRepairer
	controller      *scpd.SCPController
	controllerReady chan struct{}
	stopped         atomic.Bool
	cancel          context.CancelFunc
	ctx             context.Context
	inflight        sync.WaitGroup
}

func NewSCPBroker(
	ctx context.Context,
	router router.Router,
	storage storage.SCPEnvelopePersistence,
	repair SCPRepairer,
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

// Stop stops the broker from calling the scp controller.
// Blocks until all in-flight controller calls have completed.
func (x *SCPBroker) Stop() {
	x.stopped.CompareAndSwap(false, true)
	x.cancel()
	x.inflight.Wait()
}

func (x *SCPBroker) IsStopped() bool {
	return x.stopped.Load()
}

func (x *SCPBroker) SetSCPController(controller *scpd.SCPController) {
	x.controller = controller
	close(x.controllerReady)
}

func (x *SCPBroker) SCPWaitReady() {
	<-x.controllerReady
}

func (x *SCPBroker) SCPInitLowestNonExternalized(ctx context.Context, slot scpkit.Slot) {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		return
	}
	x.controller.SetLowestNonExternalized(ctx, slot)
}

func (x *SCPBroker) SCPReviveStateFromEnvelopes(ctx context.Context, slot scpkit.Slot) {
	// panic("XXX")
}

func (x *SCPBroker) SCPAcceptReceiveEnvelopes(ctx context.Context) {
	// panic("XXX")
}

func (x *SCPBroker) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.controllerReady
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		return
	}

	liblog.Debugf(ctx, "feeding envelope for slot %d", env.Statement.Slot_index)
	x.storage.PersistSCPEnvelope(ctx, env)
	x.controller.SCPReceivedEnvelope(ctx, env)
}

func (x *SCPBroker) RepairEnvelopes(ctx context.Context, slot proto.SCPSlotIndex) []proto.SCPEnvelope {
	return x.storage.FindSCPEnvelopesForSlot(ctx, slot)
}

func (x *SCPBroker) SCPBackfeedEnvelopes(ctx context.Context, slot scpkit.Slot, timeout time.Duration) {
	go x.fulfillBackfeed(ctx, slot, timeout)
}

func (x *SCPBroker) fulfillBackfeed(callerCtx context.Context, slot scpkit.Slot, timeout time.Duration) {
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

// impl SCPController

func (x *SCPBroker) SCPNominate(ctx context.Context, slot scpkit.Slot, value scpkit.Value, prev scpkit.Value) bool {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	return x.controller.SCPNominate(ctx, slot, value, prev)
}

func (x *SCPBroker) SCPStopNomination(ctx context.Context, slot scpkit.Slot) {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	x.controller.SCPStopNomination(ctx, slot)
}

func (x *SCPBroker) SCPUpdateQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	x.controller.SCPUpdateQuorumSet(ctx, qset)
}

func (x *SCPBroker) SCPReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	x.controller.SCPReceivedEnvelope(ctx, env)
}

func (x *SCPBroker) SCPSetKnownQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) proto.SCPOptQuorumSet {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	return x.controller.SCPSetKnownQuorumSet(ctx, qset)
}

func (x *SCPBroker) SCPGetKnownQuorumSet(ctx context.Context, qhash proto.SCPHash) proto.SCPOptQuorumSet {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	return x.controller.SCPGetKnownQuorumSet(ctx, qhash)
}

func (x *SCPBroker) SCPRemoveKnownQuorumSet(ctx context.Context, qhash proto.SCPHash) proto.SCPOptQuorumSet {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	return x.controller.SCPRemoveKnownQuorumSet(ctx, qhash)
}
