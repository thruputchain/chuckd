package dogwood

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
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
	endpointMaxPending  int           // maximum number of pending (inflight) calls to an endpoint
	endpointNumClients  int           // number of client connections to endpoint
	endpointIdleTTL     time.Duration // discard idle endpoints after ttl
	endpointRedialDelay time.Duration
	// state
	lk       sync.Mutex
	endpoint map[proto.ValidatorID]*endpoint
}

type endpoint struct {
	lastUse     time.Time // accessed by caller under caller's lock (in getEndpoint and expire)
	addr        proto.ValidatorID
	time        libtime.Time
	callWorkers libflow.WorkerPool
	client      *ClientPool
}

// caller

func NewCaller(
	ctx context.Context,
	time libtime.Time,
	dialer DogwoodDialer,
	endpointMaxPending int,
	endpointNumClients int,
	endpointIdleTTL time.Duration,
	endpointRedialDelay time.Duration,
) Caller {
	return &caller{
		ctx:                 ctx,
		time:                time,
		dialer:              dialer,
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

	return &endpoint{
		// last: set by getEndpoint; accessed under caller.lk
		addr:        addr,
		time:        x.time,
		callWorkers: libflow.NewWorkerPool(ctx, protoutil.VIDString(addr)+"-call-workers", x.endpointMaxPending),
		client:      NewClientPool(ctx, x.time, x.dialer, addr, x.endpointNumClients, x.endpointRedialDelay),
	}
}

func (x *endpoint) Stop() {
	x.callWorkers.Stop()
}

func (x *endpoint) DogwoodDeliver(ctx context.Context, arg proto.DogwoodDeliveryEnvelope) error {
	x.callWorkers.Submit(
		func() {
			err := x.client.Dogwood_deliver(ctx, arg)
			if err != nil {
				liblog.Errorf("calling deliver on endpoint %v: %v", protoutil.VIDString(x.addr), err)
			}
		},
	)
	return nil
}
