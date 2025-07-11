package dogwood

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/panjf2000/ants/v2"
)

type StreamSender struct {
	// systems
	rand      *rand.Rand
	discovery *discovery.DiscoveryService
	nav       *Navigator
	caller    Caller
	// performance
	m  xdrutil.XdrMarshaler
	wp *ants.Pool
	// state
	lk     sync.Mutex
	stream map[proto.DogwoodStream]*SendStream
}

type SendStream struct {
	parent *StreamSender
	stream proto.DogwoodStream
	route  proto.DogwoodRoute
	nav    NavRoute
}

func NewStreamSender(
	ctx context.Context,
	rand *rand.Rand,
	discovery *discovery.DiscoveryService,
	nav *Navigator,
	caller Caller,
	numWorkers int,
) *StreamSender {
	wp, err := ants.NewPool(numWorkers)
	must.NoError(ctx, err)
	return &StreamSender{
		rand:      rand,
		discovery: discovery,
		nav:       nav,
		caller:    caller,
		m:         xdrutil.NewXdrMarshaler(),
		wp:        wp,
		stream:    map[proto.DogwoodStream]*SendStream{},
	}
}

func (x *StreamSender) DialOptRoute(ctx context.Context, optRoute *dogwoodutil.LinkedTreeNode) (*SendStream, error) {
	if optRoute == nil {
		return x.Dial(ctx)
	} else {
		return x.DialRoute(ctx, optRoute)
	}
}

func (x *StreamSender) DialRoute(ctx context.Context, t *dogwoodutil.LinkedTreeNode) (*SendStream, error) {

	downstream, _ := t.FindChildren(x.discovery.SelfID())
	nav := NavRoute{
		Hash:       dogwoodutil.TreeHash(x.m, t),
		Downstream: downstream,
	}

	stream := proto.DogwoodStream{
		Source_id: x.discovery.SelfID(),
		Stream_id: rand.Int63(),
	}

	nodes := t.TreeNodes(x.m)
	route := proto.DogwoodRoute{
		Route_hash:  nil,
		Route_nodes: &nodes,
	}

	ss := x.newSendStream(ctx, stream, route, nav)

	x.lk.Lock()
	defer x.lk.Unlock()

	x.stream[stream] = ss

	return ss, nil

}

func (x *StreamSender) Dial(ctx context.Context) (*SendStream, error) {

	nav, err := x.nav.GetRoute(ctx)
	if err != nil {
		return nil, err
	}

	stream := proto.DogwoodStream{
		Source_id: x.discovery.SelfID(),
		Stream_id: rand.Int63(),
	}

	route := proto.DogwoodRoute{
		Route_hash:  &nav.Hash,
		Route_nodes: nil,
	}

	ss := x.newSendStream(ctx, stream, route, nav)

	x.lk.Lock()
	defer x.lk.Unlock()

	x.stream[stream] = ss

	return ss, nil
}

// stream

func (x *StreamSender) newSendStream(
	_ context.Context,
	stream proto.DogwoodStream,
	route proto.DogwoodRoute,
	nav NavRoute,
) *SendStream {
	return &SendStream{
		parent: x,
		stream: stream,
		route:  route,
		nav:    nav,
	}
}

func (s *SendStream) Stream() proto.DogwoodStream {
	return s.stream
}

func (x *SendStream) Send(ctx context.Context, pkg *proto.DogwoodPackage) {

	x.parent.wp.Submit(
		func() {

			delivery := proto.DogwoodDelivery{
				Stream: x.stream,
				Route:  x.route,
				Pkg:    *pkg,
			}
			env := proto.DogwoodDeliveryEnvelope{
				Signature: protoutil.SignXdr(x.parent.m, x.parent.discovery.SecretKey(), &delivery),
				Delivery:  delivery,
			}

			for _, d := range x.nav.Downstream {
				x.parent.caller.DogwoodDeliver(ctx, d, env)
			}
		},
	)
}

func (x *SendStream) Close() {
	x.parent.lk.Lock()
	defer x.parent.lk.Unlock()

	delete(x.parent.stream, x.stream)
}
