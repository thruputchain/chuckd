package dogwood

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	proto "github.com/groundhogchain/protocol-go"
)

type Caller interface {
	DogwoodRoute(context.Context, proto.ValidatorID, proto.DogwoodRouteEnvelope) error
	DogwoodDeliver(context.Context, proto.ValidatorID, []proto.DogwoodDeliveryEnvelope) error
}

type caller struct {
	ctx    context.Context
	time   libtime.Time
	dialer DogwoodDialer
	// params
	queueSize int           // number of calls in each endpoint's call queue
	idleTTL   time.Duration // discard idle endpoints after ttl
	redialTTL time.Duration
	// state
	lk       sync.Mutex
	endpoint map[proto.ValidatorID]*endpoint
}

type endpoint struct {
	lastUse   time.Time // accessed by caller under caller's lock (in getEndpoint and expire)
	addr      proto.ValidatorID
	time      libtime.Time
	dialer    DogwoodDialer
	redialTTL time.Duration
	stop      chan struct{}
	rq        chan routeCall
	dq        chan deliverCall
	failed    *time.Time // accessed synchronously in Start
}

type routeCall struct {
	stamp time.Time
	arg   proto.DogwoodRouteEnvelope
}

type deliverCall struct {
	stamp time.Time
	args  []proto.DogwoodDeliveryEnvelope
}

// caller

func NewCaller(
	ctx context.Context,
	time libtime.Time,
	dialer DogwoodDialer,
	endpointQueueSize int,
	idleEndpointTTL time.Duration,
	endpointRedialTTL time.Duration,
) Caller {
	return &caller{
		ctx:       ctx,
		time:      time,
		dialer:    dialer,
		queueSize: endpointQueueSize,
		idleTTL:   idleEndpointTTL,
		redialTTL: endpointRedialTTL,
		endpoint:  map[proto.ValidatorID]*endpoint{},
	}
}

func (x *caller) Start(ctx context.Context) {
	go x.expire(ctx) // expire endpoints which have not been used for a long time
}

func (x *caller) expire(_ context.Context) {
	for {
		x.time.Sleep(x.idleTTL)
		now := x.time.Now()
		x.lk.Lock()
		for key, e := range x.endpoint {
			if now.Sub(e.lastUse) > x.idleTTL {
				delete(x.endpoint, key)
				e.Stop()
			}
		}
		x.lk.Unlock()
	}
}

func (x *caller) getEndpoint(addr proto.ValidatorID) *endpoint {
	x.lk.Lock()
	defer x.lk.Unlock()

	e, ok := x.endpoint[addr]
	if !ok {
		e = x.newEndpoint(addr)
		go e.Start(x.ctx)
		x.endpoint[addr] = e
	}
	e.lastUse = x.time.Now() // update time to ensure no expiry for next call duration

	return e
}

func (x *caller) DogwoodRoute(ctx context.Context, addr proto.ValidatorID, arg proto.DogwoodRouteEnvelope) error {
	return x.getEndpoint(addr).DogwoodRoute(ctx, arg)
}

func (x *caller) DogwoodDeliver(ctx context.Context, addr proto.ValidatorID, args []proto.DogwoodDeliveryEnvelope) error {
	return x.getEndpoint(addr).DogwoodDeliver(ctx, args)
}

// endpoint

func (x *caller) newEndpoint(addr proto.ValidatorID) *endpoint {
	return &endpoint{
		// last: set by getEndpoint; accessed under caller.lk
		addr:      addr,
		time:      x.time,
		dialer:    x.dialer,
		redialTTL: x.redialTTL,
		rq:        make(chan routeCall, x.queueSize),
		dq:        make(chan deliverCall, x.queueSize),
		stop:      make(chan struct{}),
	}
}

func (x *endpoint) Stop() {
	close(x.stop)
}

func (x *endpoint) Start(ctx context.Context) {

	c, _ := x.dial(ctx)

	// close client on exit from this function
	defer func() {
		if c != nil {
			c.Close()
		}
	}()

	// drain rq and dq
	defer func() {
		for range x.dq {
		}
		for range x.rq {
		}
	}()

	// process calls
	for {
		select {
		case <-x.stop:
			return
		case <-ctx.Done():
			return
		case rcall := <-x.rq:
			// XXX: make client calls non-blocking
			err := c.Dogwood_route(ctx, rcall.arg)
			if err != nil {
				liblog.Errorf("calling route on endpoint %v: %v", form.Display(x.addr), err)
				c, _ = x.dial(ctx)
			}
		case dcall := <-x.dq:
			for _, arg := range dcall.args {
				err := c.Dogwood_deliver(ctx, arg)
				if err != nil {
					liblog.Errorf("calling deliver on endpoint %v: %v", form.Display(x.addr), err)
					c, _ = x.dial(ctx)
				}
			}
		}
	}
}

// if the endpoint is unreachable, makes a note, and fails all redials within a given duration
func (x *endpoint) dial(ctx context.Context) (DogwoodClientCloser, error) {
	if x.failed != nil && x.time.Now().Sub(*x.failed) < x.redialTTL {
		return nil, fmt.Errorf("recently unreachable")
	}
	c, err := x.dialer.Dial(ctx, x.addr)
	if err != nil {
		now := x.time.Now()
		x.failed = &now
		return nil, err
	}
	x.failed = nil
	return c, nil
}

func (x *endpoint) DogwoodRoute(ctx context.Context, arg proto.DogwoodRouteEnvelope) error {
	x.rq <- routeCall{stamp: x.time.Now(), arg: arg}
	return nil
}

func (x *endpoint) DogwoodDeliver(ctx context.Context, args []proto.DogwoodDeliveryEnvelope) error {
	x.dq <- deliverCall{stamp: x.time.Now(), args: args}
	return nil
}
