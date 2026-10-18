package dogwood

import (
	"context"
	"math/rand"
	"sync"

	"github.com/thruputchain/chuckd/net/discovery"
	"github.com/thruputchain/chuckd/util/dogwoodutil"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/libflow"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libmem"
	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type StreamSender struct {
	// systems
	rand      *rand.Rand
	discovery discovery.Discovery
	nav       *Navigator
	caller    Caller
	// performance
	m           libxdr.XdrMarshaler
	sendWorkers libflow.SubmitWorkerPool
	// state
	lk      sync.Mutex
	streams map[proto.DogwoodStream]*SendStream
}

type SendStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	parent *StreamSender
	stream proto.DogwoodStream
	route  proto.DogwoodRoute
	nav    NavRoute
}

func NewStreamSender(
	ctx context.Context,
	rand *rand.Rand,
	discovery discovery.Discovery,
	nav *Navigator,
	caller Caller,
	numWorkers int,
) *StreamSender {

	return &StreamSender{
		rand:        rand,
		discovery:   discovery,
		nav:         nav,
		caller:      caller,
		m:           libxdr.NewXdrMarshaler(),
		sendWorkers: libflow.NewSubmitWorkerPool(ctx, numWorkers),
		streams:     map[proto.DogwoodStream]*SendStream{},
	}
}

func (x *StreamSender) NumStreams() int {
	x.lk.Lock()
	defer x.lk.Unlock()
	return len(x.streams)
}

func (x *StreamSender) Start(context.Context) {}

func (x *StreamSender) DialOptRoute(ctx context.Context, optRoute *dogwoodutil.LinkedDAGNode) (*SendStream, error) {
	if optRoute == nil {
		return x.Dial(ctx)
	} else {
		return x.DialRoute(ctx, optRoute)
	}
}

func (x *StreamSender) DialRoute(ctx context.Context, t *dogwoodutil.LinkedDAGNode) (*SendStream, error) {

	downstream, _ := t.FindChildren(x.discovery.SelfID())
	nav := NavRoute{
		Hash:       dogwoodutil.TreeHash(x.m, t),
		Downstream: downstream,
	}

	stream := proto.DogwoodStream{
		Source_id: x.discovery.SelfID(),
		Stream_id: rand.Uint64(),
	}

	nodes := t.TreeNodes(x.m)
	route := proto.DogwoodRoute{
		Route_hash:  nil,
		Route_nodes: &nodes,
	}

	ss := x.newSendStream(ctx, stream, route, nav)

	x.lk.Lock()
	defer x.lk.Unlock()

	x.streams[stream] = ss

	return ss, nil

}

func (x *StreamSender) Dial(ctx context.Context) (*SendStream, error) {

	nav, err := x.nav.GetRoute(ctx)
	if err != nil {
		return nil, err
	}

	stream := proto.DogwoodStream{
		Source_id: x.discovery.SelfID(),
		Stream_id: rand.Uint64(),
	}

	route := proto.DogwoodRoute{
		Route_hash:  &nav.Hash,
		Route_nodes: nil,
	}

	ss := x.newSendStream(ctx, stream, route, nav)

	x.lk.Lock()
	defer x.lk.Unlock()

	x.streams[stream] = ss

	return ss, nil
}

// stream

func (x *StreamSender) newSendStream(
	ctx context.Context,
	stream proto.DogwoodStream,
	route proto.DogwoodRoute,
	nav NavRoute,
) *SendStream {
	ctx, cancel := context.WithCancel(ctx)
	return &SendStream{
		ctx:    ctx,
		cancel: cancel,
		parent: x,
		stream: stream,
		route:  route,
		nav:    nav,
	}
}

func (s *SendStream) Stream() proto.DogwoodStream {
	return s.stream
}

func (x *SendStream) Send(ctx context.Context, pkg libmem.BorrowedValue[*proto.DogwoodPackage]) {

	x.parent.sendWorkers.Submit(
		func() {

			delivery := proto.DogwoodDelivery{
				Stream: x.stream,
				Route:  x.route,
				Pkg:    *pkg.Value(),
			}
			env := proto.DogwoodDeliveryEnvelope{
				Signature: protoutil.SignXdr(x.parent.m, x.parent.discovery.SecretKey(), &delivery),
				Delivery:  delivery,
			}

			var wg sync.WaitGroup
			wg.Add(len(x.nav.Downstream))

			for _, d := range x.nav.Downstream {
				err := x.parent.caller.DogwoodDeliver(ctx, d, libmem.BorrowOnWaitGroup(&wg, env))
				if err != nil {
					liblog.Infof(ctx, "sending delivery: %v", err)
				}
			}

			go func() {
				wg.Wait()
				pkg.Recycle()
			}()
		},
	)
}

func (x *SendStream) Close() {
	x.cancel()

	x.parent.lk.Lock()
	defer x.parent.lk.Unlock()

	delete(x.parent.streams, x.stream)
}

// stats

type StreamSenderStats struct {
	NumOpenStreams int
	libflow.WorkerPoolStats
}

func (s *StreamSender) Stats() StreamSenderStats {
	s.lk.Lock()
	n := len(s.streams)
	s.lk.Unlock()
	return StreamSenderStats{
		NumOpenStreams:  n,
		WorkerPoolStats: s.sendWorkers.Stats(),
	}
}
