package dogwood

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libmem"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Caller interface {
	Start(ctx context.Context)
	DogwoodDeliver(context.Context, proto.ValidatorID, libmem.BorrowedValue[proto.DogwoodDeliveryEnvelope]) error
	Stats() CallerStats
}

type caller struct {
	ctx    context.Context
	time   libtime.Time
	dialer DogwoodDialer
	// params
	endpointMaxPending  int           // maximum number of pending (inflight) calls to an endpoint
	endpointNumClients  int           // number of client connections to endpoint
	endpointIdleTTL     time.Duration // discard idle endpoints after ttl
	endpointDialTimeout time.Duration
	endpointRedialDelay time.Duration
	endpointCallTimeout time.Duration
	// state
	lk        sync.Mutex
	endpoints map[proto.ValidatorID]*endpoint
}

type endpoint struct {
	parent      *caller
	lastUse     time.Time // accessed by caller under caller's lock (in getEndpoint and expire)
	addr        proto.ValidatorID
	time        libtime.Time
	callWorkers libflow.SubmitWorkerPool
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
	endpointDialTimeout time.Duration,
	endpointRedialDelay time.Duration,
	endpointCallTimeout time.Duration,
) Caller {
	return &caller{
		ctx:                 ctx,
		time:                time,
		dialer:              dialer,
		endpointMaxPending:  endpointMaxPending,
		endpointNumClients:  endpointNumClients,
		endpointIdleTTL:     endpointIdleTTL,
		endpointDialTimeout: endpointDialTimeout,
		endpointRedialDelay: endpointRedialDelay,
		endpointCallTimeout: endpointCallTimeout,
		endpoints:           map[proto.ValidatorID]*endpoint{},
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
		for key, e := range x.endpoints {
			if now.Sub(e.lastUse) > x.endpointIdleTTL {
				delete(x.endpoints, key)
				e.Stop()
			}
		}
		x.lk.Unlock()
	}
}

func (x *caller) getEndpoint(ctx context.Context, addr proto.ValidatorID) *endpoint {
	x.lk.Lock()
	defer x.lk.Unlock()

	e, ok := x.endpoints[addr]
	if !ok {
		e = x.newEndpoint(ctx, addr)
		x.endpoints[addr] = e
	}
	e.lastUse = x.time.Now() // update time to ensure no expiry for next call duration

	return e
}

func (x *caller) DogwoodDeliver(
	ctx context.Context,
	addr proto.ValidatorID,
	arg libmem.BorrowedValue[proto.DogwoodDeliveryEnvelope],
) error {
	return x.getEndpoint(ctx, addr).DogwoodDeliver(ctx, arg)
}

// endpoint

func (x *caller) newEndpoint(ctx context.Context, addr proto.ValidatorID) *endpoint {

	return &endpoint{
		parent: x,
		// last: set by getEndpoint; accessed under caller.lk
		addr:        addr,
		time:        x.time,
		callWorkers: libflow.NewSubmitWorkerPool(ctx, x.endpointMaxPending),
		client:      NewClientPool(ctx, x.time, x.dialer, addr, x.endpointNumClients, x.endpointDialTimeout, x.endpointRedialDelay),
	}
}

func (x *endpoint) Stop() {
	x.callWorkers.Stop()
}

func (x *endpoint) DogwoodDeliver(
	ctx context.Context,
	arg libmem.BorrowedValue[proto.DogwoodDeliveryEnvelope],
) error {
	x.callWorkers.Submit(
		func() {
			defer arg.Recycle()
			ctx, cancel := context.WithTimeout(ctx, x.parent.endpointCallTimeout)
			defer cancel()
			err := x.client.Dogwood_deliver(ctx, arg.Value())
			if err != nil {
				liblog.Errorf("calling deliver on endpoint %v: %v", protoutil.VIDString(x.addr), err)
			}
		},
	)
	return nil
}

// stats

type CallerStats struct {
	Endpoints []EndpointStats
}

func (x CallerStats) Summarize() string {
	var w bytes.Buffer
	for _, e := range x.Endpoints {
		fmt.Fprintf(&w, "%s ", e.Summarize())
	}
	return w.String()
}

type EndpointStats struct {
	CallWorkers libflow.WorkerPoolStats
	ClientPoolStats
}

func (x EndpointStats) Summarize() string {
	return fmt.Sprintf("(util=%.2f%%, lat=%s)", x.CallWorkers.Utilization.Average*100, x.DeliverLat.Summarize())
}

func (x *caller) Stats() CallerStats {
	x.lk.Lock()
	defer x.lk.Unlock()
	es := make([]EndpointStats, 0, len(x.endpoints))
	for _, e := range x.endpoints {
		es = append(es, e.Stats())
	}
	return CallerStats{
		Endpoints: es,
	}
}

func (x *endpoint) Stats() EndpointStats {
	return EndpointStats{
		CallWorkers:     x.callWorkers.Stats(),
		ClientPoolStats: x.client.Stats(),
	}
}
