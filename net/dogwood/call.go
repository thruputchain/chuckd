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
	DogwoodDeliver(context.Context, proto.ValidatorID, proto.DogwoodDeliveryEnvelope) error
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
	dq        chan deliverCall
	failed    *time.Time // accessed synchronously in Start
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

func (x *caller) DogwoodDeliver(ctx context.Context, addr proto.ValidatorID, arg proto.DogwoodDeliveryEnvelope) error {
	return x.getEndpoint(addr).DogwoodDeliver(ctx, arg)
}

// endpoint

func (x *caller) newEndpoint(addr proto.ValidatorID) *endpoint {
	return &endpoint{
		// last: set by getEndpoint; accessed under caller.lk
		addr:      addr,
		time:      x.time,
		dialer:    x.dialer,
		redialTTL: x.redialTTL,
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

	eq := make(chan error)

	// drain rq and dq
	defer func() {
		for range x.dq {
		}
		for range eq {
		}
	}()

	// process calls
	for {
		select {
		case <-x.stop:
			return
		case <-ctx.Done():
			return
		case <-eq:
			c, _ = x.dial(ctx)
		case dcall := <-x.dq:
			// XXX: support discard-reply semantics in goxdr driver
			go func() {
				err := c.Dogwood_deliver(ctx, dcall.arg)
				if err != nil {
					liblog.Errorf("calling deliver on endpoint %v: %v", form.Display(x.addr), err)
					eq <- err
				}
			}()
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

func (x *endpoint) DogwoodDeliver(ctx context.Context, arg proto.DogwoodDeliveryEnvelope) error {
	x.dq <- deliverCall{stamp: x.time.Now(), arg: arg}
	return nil
}
