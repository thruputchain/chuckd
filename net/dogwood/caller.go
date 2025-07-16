package dogwood

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/panjf2000/ants/v2"
)

type Caller interface {
	Start(ctx context.Context)
	DogwoodDeliver(context.Context, proto.ValidatorID, proto.DogwoodDeliveryEnvelope) error
}

type caller struct {
	ctx    context.Context
	time   libtime.Time
	dialer DogwoodDialer
	// params
	endpointQueueLen    int           // maximum number of calls in endpoint call queue
	endpointMaxPending  int           // maximum number of pending (inflight) calls to an endpoint
	endpointNumClients  int           // number of client connections to endpoint
	endpointIdleTTL     time.Duration // discard idle endpoints after ttl
	endpointRedialDelay time.Duration
	// state
	lk       sync.Mutex
	endpoint map[proto.ValidatorID]*endpoint
}

type endpoint struct {
	lastUse time.Time // accessed by caller under caller's lock (in getEndpoint and expire)
	addr    proto.ValidatorID
	time    libtime.Time
	// dialer      DogwoodDialer
	// redialDelay time.Duration
	stop    chan struct{}
	dq      chan deliverCall
	pending *ants.Pool
	// failed      *time.Time // accessed synchronously in endpoint.Start
	client DogwoodClientCloser
}

type deliverCall struct {
	stamp time.Time
	arg   proto.DogwoodDeliveryEnvelope
}

// caller

func NewCaller(
	ctx context.Context,
	time libtime.Time,
	dialer DogwoodDialer,
	endpointQueueLen int,
	endpointMaxPending int,
	endpointNumClients int,
	endpointIdleTTL time.Duration,
	endpointRedialDelay time.Duration,
) Caller {
	return &caller{
		ctx:                 ctx,
		time:                time,
		dialer:              dialer,
		endpointQueueLen:    endpointQueueLen,
		endpointMaxPending:  endpointMaxPending,
		endpointNumClients:  endpointNumClients,
		endpointIdleTTL:     endpointIdleTTL,
		endpointRedialDelay: endpointRedialDelay,
		endpoint:            map[proto.ValidatorID]*endpoint{},
	}
}

func (x *caller) Start(ctx context.Context) {
	go x.expire(ctx) // expire endpoints which have not been used for a long time
}

func (x *caller) expire(_ context.Context) {
	for {
		x.time.Sleep(x.endpointIdleTTL)
		now := x.time.Now()
		x.lk.Lock()
		for key, e := range x.endpoint {
			if now.Sub(e.lastUse) > x.endpointIdleTTL {
				delete(x.endpoint, key)
				e.Stop()
			}
		}
		x.lk.Unlock()
	}
}

func (x *caller) getEndpoint(ctx context.Context, addr proto.ValidatorID) *endpoint {
	x.lk.Lock()
	defer x.lk.Unlock()

	e, ok := x.endpoint[addr]
	if !ok {
		e = x.newEndpoint(ctx, addr)
		go e.Start(x.ctx)
		x.endpoint[addr] = e
	}
	e.lastUse = x.time.Now() // update time to ensure no expiry for next call duration

	return e
}

func (x *caller) DogwoodDeliver(ctx context.Context, addr proto.ValidatorID, arg proto.DogwoodDeliveryEnvelope) error {
	return x.getEndpoint(ctx, addr).DogwoodDeliver(ctx, arg)
}

// endpoint

func (x *caller) newEndpoint(ctx context.Context, addr proto.ValidatorID) *endpoint {

	pending, err := ants.NewPool(x.endpointMaxPending)
	must.NoError(ctx, err)
	return &endpoint{
		// last: set by getEndpoint; accessed under caller.lk
		addr:    addr,
		time:    x.time,
		dq:      make(chan deliverCall, x.endpointQueueLen),
		pending: pending,
		stop:    make(chan struct{}),
		client:  NewClientPool(ctx, x.time, x.dialer, addr, x.endpointNumClients, x.endpointRedialDelay),
	}
}

func (x *endpoint) Stop() {
	close(x.stop)
}

func (x *endpoint) Start(ctx context.Context) {

	defer x.client.Close()

	defer func() {
		for range x.dq {
		}
	}()

	for {
		select {
		case <-x.stop:
			return
		case <-ctx.Done():
			return
		case dcall := <-x.dq:
			x.pending.Submit(
				func() {
					err := x.client.Dogwood_deliver(ctx, dcall.arg)
					if err != nil {
						liblog.Errorf("calling deliver on endpoint %v: %v", protoutil.VIDString(x.addr), err)
					}
				},
			)
		}
	}
}

func (x *endpoint) DogwoodDeliver(ctx context.Context, arg proto.DogwoodDeliveryEnvelope) error {
	x.dq <- deliverCall{stamp: x.time.Now(), arg: arg}
	return nil
}
