package dogwood

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libflow"
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
	proc StreamProcessor
	// systems
	discovery discovery.Discovery
	nav       *Navigator
	caller    Caller
	// performance
	m              xdrutil.XdrMarshaler
	receiveWorkers libflow.WorkerPool
	// state
	lk      sync.Mutex
	streams map[proto.DogwoodStream]*ReceiveStream
}

type ReceiveStream struct {
	parent *StreamReceiver
	stream proto.DogwoodStream
	nav    NavRoute
}

func NewStreamReceiver(
	ctx context.Context,
	discovery discovery.Discovery,
	nav *Navigator,
	caller Caller,
	numWorkers int,
) *StreamReceiver {

	return &StreamReceiver{
		discovery:      discovery,
		nav:            nav,
		caller:         caller,
		m:              xdrutil.NewXdrMarshaler(),
		receiveWorkers: libflow.NewWorkerPool(ctx, "stream-receive-workers", numWorkers),
		streams:        map[proto.DogwoodStream]*ReceiveStream{},
	}
}

func (r *StreamReceiver) SetProcessor(proc StreamProcessor) {
	r.proc = proc
}

func (r *StreamReceiver) NumStreams() int {
	r.lk.Lock()
	defer r.lk.Unlock()
	return len(r.streams)
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

	r.lk.Lock()
	s, ok := r.streams[env.Delivery.Stream]
	if !ok {
		s = r.newReceivingStream(ctx, env.Delivery.Stream, route)
		r.streams[env.Delivery.Stream] = s
	}
	r.lk.Unlock()

	s.process(ctx, env)
}

// stream

func (r *StreamReceiver) newReceivingStream(
	ctx context.Context,
	stream proto.DogwoodStream,
	route NavRoute,
) *ReceiveStream {
	return &ReceiveStream{
		parent: r,
		stream: stream,
		nav:    route,
	}
}

func (s *ReceiveStream) Close() {

	s.parent.lk.Lock()
	defer s.parent.lk.Unlock()
	delete(s.parent.streams, s.stream)
}

func (s *ReceiveStream) process(ctx context.Context, env proto.DogwoodDeliveryEnvelope) {

	s.parent.proc.OnReceive(s.stream.Source_id, &env.Delivery.Pkg)

	for _, d := range s.nav.Downstream {
		s.parent.caller.DogwoodDeliver(ctx, d, env)
	}
}

// stats

type StreamReceiverStats struct {
	NumOpenStreams int
	libflow.WorkerPoolStats
}

func (s *StreamReceiver) Stats() StreamReceiverStats {
	s.lk.Lock()
	n := len(s.streams)
	s.lk.Unlock()
	return StreamReceiverStats{
		NumOpenStreams:  n,
		WorkerPoolStats: s.receiveWorkers.Stats(),
	}
}
