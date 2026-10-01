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
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
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
	feedEnvelopes   chan struct{}
	stopped         atomic.Bool
	cancel          context.CancelFunc
	ctx             context.Context
	inflight        sync.WaitGroup
	m               libxdr.XdrMarshaler
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
		feedEnvelopes:   make(chan struct{}),
		cancel:          cancel,
		ctx:             brokerCtx,
		m:               libxdr.NewXdrMarshaler(),
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

func (x *SCPBroker) SCPReviveStateFromEnvelopes(ctx context.Context, slot scpkit.Slot) error {
	slots, err := x.storage.ListSlotsWithSCPEnvelopesNotBelow(ctx, slot)
	if err != nil {
		return err
	}
	for _, slot := range slots {
		liblog.Infof(ctx, "reviving state from envelopes for slot %d", slot)
		envs, err := x.storage.FindSCPEnvelopesForSlotFromLocal(ctx, slot)
		if err != nil {
			return err
		}
		for _, env := range envs {
			if protoutil.NodeToValidatorID(env.Statement.Node_id) == protoutil.PublicVID(x.router.Self()) {
				liblog.Infof(ctx, "scp set state from envelope: slot=%d; env=%v",
					slot, protoutil.SCPEnvelopeDisplay(env))
				x.SCPSetStateFromEnvelope(ctx, slot, env)
			}
		}
	}
	return nil
}

func (x *SCPBroker) SCPAllowFeedingEnvelopes(ctx context.Context) {
	close(x.feedEnvelopes)
}

// SCPFeedEnvelope feeds an envelope to the scp controller.
// It verifies the signature of the envelope and persists it to the storage.
// It then calls the scp controller to handle the envelope.
// SECURITY: SCPFeedEnvelope is the only entrypoint for envelopes coming from
// the network into the system.
func (x *SCPBroker) SCPFeedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	<-x.controllerReady
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		return
	}

	liblog.Debugf(ctx, "feeding envelope for slot %d", env.Statement.Slot_index)

	if err := scpkit.VerifyEnvelope(x.m, &env); err != nil {
		liblog.Infof(ctx, "dropping envelope due to %v: %v", err, protoutil.SCPEnvelopeDisplay(env))
		return
	}

	x.storage.PersistSCPEnvelope(ctx, env)
	x.controller.SCPReceivedEnvelope(ctx, env)
}

func (x *SCPBroker) RepairEnvelopes(ctx context.Context, slot proto.SCPSlotIndex) ([]proto.SCPEnvelope, error) {
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
		if env.Statement.Node_id.Scheme != proto.Ed25519 {
			liblog.Infof(ctx, "dropping backfilled envelope: unsupported public key scheme %d", env.Statement.Node_id.Scheme)
			continue
		}
		liblog.Debugf(ctx, "backfilling envelope %d of %d for slot %d", i+1, len(envs), env.Statement.Slot_index)
		// replay envelopes for the requested slot, but not our own!
		if env.Statement.Slot_index == slot &&
			!protoutil.ValidatorIDEqual(
				protoutil.NodeToValidatorID(env.Statement.Node_id), protoutil.PublicVID(x.router.Self())) {
			x.SCPFeedEnvelope(ctx, env)
		}
	}
}

// impl SCPController

func (x *SCPBroker) SCPSetLowestNonExternalized(ctx context.Context, slot scpkit.Slot) {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	x.controller.SCPSetLowestNonExternalized(ctx, slot)
}

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
	// do not let envelopes through until allowed by SCPAcceptReceiveEnvelopes
	<-x.feedEnvelopes

	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	x.controller.SCPReceivedEnvelope(ctx, env)
}

func (x *SCPBroker) SCPSetStateFromEnvelope(ctx context.Context, slot scpkit.Slot, env proto.SCPEnvelope) {
	x.inflight.Add(1)
	defer x.inflight.Done()
	if x.IsStopped() {
		panic("SCPBroker stopped")
	}
	x.controller.SCPSetStateFromEnvelope(ctx, slot, env)
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
