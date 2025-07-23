package dogwood

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type StreamProcessor interface {
	OnReceive(proto.ValidatorID, *proto.DogwoodPackage)
}

type StreamReceiver struct {
	time libtime.Time
	proc StreamProcessor
	// params
	idleTTL time.Duration
	// systems
	discovery discovery.Discovery
	nav       *Navigator
	caller    Caller
	// performance
	m              xdrutil.XdrMarshaler
	receiveWorkers libflow.SubmitWorkerPool
	// state
}

func NewStreamReceiver(
	ctx context.Context,
	time libtime.Time,
	discovery discovery.Discovery,
	nav *Navigator,
	caller Caller,
	numWorkers int,
) *StreamReceiver {

	return &StreamReceiver{
		time:           time,
		discovery:      discovery,
		nav:            nav,
		caller:         caller,
		m:              xdrutil.NewXdrMarshaler(),
		receiveWorkers: libflow.NewSubmitWorkerPool(ctx, numWorkers),
	}
}

func (r *StreamReceiver) SetProcessor(proc StreamProcessor) {
	r.proc = proc
}

func (r *StreamReceiver) Start(ctx context.Context) {}

func (r *StreamReceiver) Deliver(ctx context.Context, env proto.DogwoodDeliveryEnvelope) {
	r.receiveWorkers.Submit(func() { r.process(ctx, env) })
}

func (r *StreamReceiver) process(ctx context.Context, env proto.DogwoodDeliveryEnvelope) {

	// recover route
	var route NavRoute

	switch {
	case env.Delivery.Route.Route_nodes != nil:
		t, err := dogwoodutil.BuildLinkedTree(r.m, *env.Delivery.Route.Route_nodes)
		if err != nil {
			break
		}
		downstream, ok := t.FindChildren(r.discovery.SelfID())
		if !ok {
			break
		}
		route = NavRoute{Hash: dogwoodutil.TreeHash(r.m, t), Downstream: downstream}
	case env.Delivery.Route.Route_hash != nil:
		route, _ = r.nav.LookupRoute(ctx, *env.Delivery.Route.Route_hash)
	}

	// verify signature, if coming from a trusted validator
	// XXX: what should we do with signatures from non-direct contact validators?
	if v, ok := r.discovery.LookupTrustedValidator(env.Delivery.Stream.Source_id); ok {
		if !protoutil.VerifyXdr(r.m, v.Public_key, &env.Delivery, env.Signature) {
			return
		}
	}

	r.proc.OnReceive(env.Delivery.Stream.Source_id, &env.Delivery.Pkg)

	for _, d := range route.Downstream {
		r.caller.DogwoodDeliver(ctx, d, env)
	}
}

// stats

type StreamReceiverStats struct {
	libflow.WorkerPoolStats
}

func (s *StreamReceiver) Stats() StreamReceiverStats {
	return StreamReceiverStats{
		WorkerPoolStats: s.receiveWorkers.Stats(),
	}
}
