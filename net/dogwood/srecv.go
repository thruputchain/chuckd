package dogwood

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/panjf2000/ants/v2"
)

type StreamReceiver struct {
	// systems
	discovery discovery.Discovery
	nav       *Navigator
	caller    Caller
	// performance
	m  xdrutil.XdrMarshaler
	wp *ants.Pool
	// flow
	deliveryQ chan proto.DogwoodDeliveryEnvelope
	acceptQ   chan *ReceiveStream
	// state
	lk     sync.Mutex
	stream map[proto.DogwoodStream]*ReceiveStream
}

type ReceiveStream struct {
	parent *StreamReceiver
	stream proto.DogwoodStream
	nav    NavRoute
	pkgQ   chan *proto.DogwoodPackage
}

func NewStreamReceiver(
	ctx context.Context,
	discovery discovery.Discovery,
	nav *Navigator,
	caller Caller,
	receiveQueueSize int,
	numWorkers int,
) *StreamReceiver {
	wp, err := ants.NewPool(numWorkers)
	must.NoError(ctx, err)
	return &StreamReceiver{
		discovery: discovery,
		nav:       nav,
		caller:    caller,
		m:         xdrutil.NewXdrMarshaler(),
		wp:        wp,
		deliveryQ: make(chan proto.DogwoodDeliveryEnvelope, receiveQueueSize),
		acceptQ:   make(chan *ReceiveStream),
		stream:    map[proto.DogwoodStream]*ReceiveStream{},
	}
}

func (r *StreamReceiver) Start(ctx context.Context) {
	go func() {
		for env := range r.deliveryQ {
			r.wp.Submit(func() { r.process(ctx, env) })
		}
	}()
}

func (r *StreamReceiver) Accept(ctx context.Context) (*ReceiveStream, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case x := <-r.acceptQ:
		return x, nil
	}
}

func (r *StreamReceiver) Deliver(ctx context.Context, env proto.DogwoodDeliveryEnvelope) {
	r.deliveryQ <- env
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
	s, ok := r.stream[env.Delivery.Stream]
	if !ok {
		s = r.newReceivingStream(env.Delivery.Stream, route)
		r.stream[env.Delivery.Stream] = s
	}
	r.lk.Unlock()

	if !ok {
		r.acceptQ <- s
	}

	s.process(ctx, env)
}

// stream

func (r *StreamReceiver) newReceivingStream(
	stream proto.DogwoodStream,
	route NavRoute,
) *ReceiveStream {
	return &ReceiveStream{
		parent: r,
		stream: stream,
		nav:    route,
		pkgQ:   make(chan *proto.DogwoodPackage),
	}
}

func (s *ReceiveStream) Close() {

	s.parent.lk.Lock()
	defer s.parent.lk.Unlock()

	delete(s.parent.stream, s.stream)
}

func (s *ReceiveStream) Stream() proto.DogwoodStream {
	return s.stream
}

func (s *ReceiveStream) Receive(ctx context.Context) (*proto.DogwoodPackage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case x := <-s.pkgQ:
		return x, nil
	}
}

func (s *ReceiveStream) process(ctx context.Context, env proto.DogwoodDeliveryEnvelope) {

	s.pkgQ <- &env.Delivery.Pkg

	for _, d := range s.nav.Downstream {
		s.parent.caller.DogwoodDeliver(ctx, d, env)
	}
}
