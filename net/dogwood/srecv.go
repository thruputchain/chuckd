package dogwood

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libflow"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libmem"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type StreamProcessor interface {
	VerifyPackage(proto.ValidatorID, *proto.DogwoodPackage) error
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
	m              libxdr.XdrMarshaler
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
		m:              libxdr.NewXdrMarshaler(),
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

	//XXX(DoS): slash the source?
	if err := r.proc.VerifyPackage(env.Delivery.Stream.Source_id, &env.Delivery.Pkg); err != nil {
		liblog.Errorf(ctx, "dropping delivery due to invalid package: %v", err)
		return
	}

	// recover route
	var route NavRoute

	switch {
	case env.Delivery.Route.Route_nodes != nil:
		t, err := dogwoodutil.BuildLinkedDagExpectTree(r.m, *env.Delivery.Route.Route_nodes)
		if err != nil {
			liblog.Errorf(ctx, "received delivery with faulty route: %v", err)
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

	// verify signature, if coming from a known validator
	// XXX(SECURITY): should we verify regardless who its from?
	if v, ok := r.discovery.LookupValidator(env.Delivery.Stream.Source_id); ok {
		if !protoutil.VerifyXdr(r.m, v.Public_key, &env.Delivery, env.Signature) {
			return
		}
	}

	r.proc.OnReceive(env.Delivery.Stream.Source_id, &env.Delivery.Pkg)

	var wg sync.WaitGroup
	wg.Add(len(route.Downstream))

	for _, d := range route.Downstream {
		err := r.caller.DogwoodDeliver(ctx, d, libmem.BorrowOnWaitGroup(&wg, env))
		if err != nil {
			liblog.Infof(ctx, "forwarding delivery: %v", err)
		}
	}

	// wg.Wait()
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
