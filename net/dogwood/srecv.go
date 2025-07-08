package dogwood

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/panjf2000/ants/v2"
)

type StreamReceiver struct {
	// systems
	discovery *discovery.DiscoveryService
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
	route  NavRoute
	pkgQ   chan *proto.DogwoodPackage
}

func NewStreamReceiver(
	ctx context.Context,
	discovery *discovery.DiscoveryService,
	nav *Navigator,
	caller Caller,
	receiveQueueSize int,
	verifyPoolSize int,
) *StreamReceiver {
	wp, err := ants.NewPool(verifyPoolSize)
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
	var err error
	switch {
	case env.Delivery.Route.Route_nodes != nil:
		t, err := dogwoodutil.BuildLinkedTree(r.m, *env.Delivery.Route.Route_nodes)
		if err != nil {
			return
		}
		downstream, ok := t.FindChildren(r.discovery.SelfID())
		if !ok {
			return
		}
		route = NavRoute{Hash: dogwoodutil.TreeHash(r.m, t), Downstream: downstream}
	case env.Delivery.Route.Route_hash != nil:
		route, err = r.nav.LookupRoute(ctx, *env.Delivery.Route.Route_hash)
		if err != nil {
			return
		}
	default:
		return
	}

	// verify signature, if coming from a trusted validator
	// XXX: what should we do with signatures from non-direct contact validators?
	if v, ok := r.discovery.LookupTrustedValidator(env.Delivery.Stream.Source_id); ok {
		if !dogwoodutil.VerifyDelivery(r.m, env, v.Public_key) {
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
		route:  route,
		pkgQ:   make(chan *proto.DogwoodPackage),
	}
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

	for _, d := range s.route.Downstream {
		s.parent.caller.DogwoodDeliver(ctx, d, env)
	}
}
