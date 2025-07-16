package dogwood

import (
	"context"
	"fmt"
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
	dialer      DogwoodDialer
	redialDelay time.Duration
	stop        chan struct{}
	dq          chan deliverCall
	pending     *ants.Pool
	failed      *time.Time // accessed synchronously in endpoint.Start
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
	endpointIdleTTL time.Duration,
	endpointRedialDelay time.Duration,
) Caller {
	return &caller{
		ctx:                 ctx,
		time:                time,
		dialer:              dialer,
		endpointQueueLen:    endpointQueueLen,
		endpointMaxPending:  endpointMaxPending,
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
		addr:        addr,
		time:        x.time,
		dialer:      x.dialer,
		redialDelay: x.endpointRedialDelay,
		dq:          make(chan deliverCall, x.endpointQueueLen),
		pending:     pending,
		stop:        make(chan struct{}),
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
			if c == nil {
				c, _ = x.dial(ctx)
			}
			// XXX: support discard-reply semantics in goxdr driver???
			if c != nil {

				cc := c
				x.pending.Submit(
					func() {
						err := cc.Dogwood_deliver(ctx, dcall.arg)
						if err != nil {
							liblog.Errorf("calling deliver on endpoint %v: %v", protoutil.VIDString(x.addr), err)
							eq <- err
						}
					},
				)

				// go func(c DogwoodClientCloser) {
				// err := c.Dogwood_deliver(ctx, dcall.arg)
				// if err != nil {
				// 	liblog.Errorf("calling deliver on endpoint %v: %v", protoutil.VIDString(x.addr), err)
				// 	eq <- err
				// }
				// }(c)
			} else {
				liblog.Errorf("dogwood call: dropping delivery to %v (no connection)", protoutil.VIDString(x.addr))
			}
		}
	}
}

// if the endpoint is unreachable, makes a note, and fails all redials within a given duration
func (x *endpoint) dial(ctx context.Context) (DogwoodClientCloser, error) {
	if x.failed != nil && x.time.Now().Sub(*x.failed) < x.redialDelay {
		liblog.Infof("dogwood dial: %v recently unreachable, not dialing", protoutil.VIDString(x.addr))
		return nil, fmt.Errorf("recently unreachable")
	}
	c, err := x.dialer.Dial(ctx, x.addr)
	if err != nil {
		now := x.time.Now()
		x.failed = &now
		liblog.Errorf("dogwood dial: %v dialing (%v)", protoutil.VIDString(x.addr), err)
		return nil, err
	}
	x.failed = nil
	return c, nil
}

func (x *endpoint) DogwoodDeliver(ctx context.Context, arg proto.DogwoodDeliveryEnvelope) error {
	x.dq <- deliverCall{stamp: x.time.Now(), arg: arg}
	return nil
}
