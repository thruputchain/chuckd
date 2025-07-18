package dogwood

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type ClientPool struct {
	address      proto.ValidatorID
	time         libtime.Time
	dialer       DogwoodDialer
	maxClients   int
	redialDelay  time.Duration
	lk           sync.Mutex
	lastDialFail *time.Time
	client       map[DogwoodClientCloser]struct{}
	//
	getClientLat *libstat.LatencyMeter
	deliverLat   *libstat.LatencyMeter
}

func NewClientPool(
	ctx context.Context,
	time_ libtime.Time,
	dialer DogwoodDialer,
	address proto.ValidatorID,
	maxClients int,
	redialDelay time.Duration,
) *ClientPool {
	return &ClientPool{
		address:     address,
		time:        time_,
		dialer:      dialer,
		maxClients:  maxClients,
		redialDelay: redialDelay,
		client:      map[DogwoodClientCloser]struct{}{},
		//
		getClientLat: libstat.NewLatencyWithReport(time_, "caller.getClient latency", 3*time.Second),
		deliverLat:   libstat.NewLatencyWithReport(time_, "caller.Dogwood_deliver latency", 3*time.Second),
	}
}

func (x *ClientPool) NumClients() int {
	x.lk.Lock()
	defer x.lk.Unlock()
	if x.client == nil {
		return 0
	}
	return len(x.client)
}

func (x *ClientPool) getClient(ctx context.Context) (client DogwoodClientCloser, err error) {

	span := x.getClientLat.Start()
	defer span.Stop()

	// get a client from the pool, if it is not closed and full
	x.lk.Lock()
	closed := x.client == nil
	full := len(x.client) >= x.maxClients
	if !closed && full {
		for c := range x.client {
			client = c
			break
		}
	}
	x.lk.Unlock()

	if closed {
		return nil, fmt.Errorf("dogwood client pool: closed")
	}

	// pool was not closed and full, returning client from pool
	if client != nil {
		return client, nil
	}

	// pool was not closed and not full, so dialing a new client
	client, err = x.dial(ctx)
	if err != nil {
		return client, err
	}

	// place the new client in the pool, if it is not closed
	x.lk.Lock()
	closed = x.client == nil
	if !closed {
		x.client[client] = struct{}{}
	}
	x.lk.Unlock()

	if closed {
		client.Close()
		return nil, fmt.Errorf("dogwood client pool: closed")
	}

	return client, nil
}

func (x *ClientPool) getLastDialFail() *time.Time {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.lastDialFail
}

func (x *ClientPool) setLastDialFail(v *time.Time) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.lastDialFail = v
}

// if the endpoint is unreachable, makes a note, and fails all redials within a given duration
func (x *ClientPool) dial(ctx context.Context) (DogwoodClientCloser, error) {

	lastDialFail := x.getLastDialFail()

	if lastDialFail != nil && x.time.Now().Sub(*lastDialFail) < x.redialDelay {
		liblog.Infof("dogwood dial: %v recently unreachable, not dialing", protoutil.VIDString(x.address))
		return nil, fmt.Errorf("recently unreachable")
	}

	c, err := x.dialer.Dial(ctx, x.address)
	if err != nil {
		now := x.time.Now()
		x.setLastDialFail(&now)
		liblog.Errorf("dogwood dial: %v dialing (%v)", protoutil.VIDString(x.address), err)
		return nil, err
	}

	x.setLastDialFail(nil)
	return c, nil
}

func (x *ClientPool) Close() {
	x.lk.Lock()
	client := x.client
	x.client = nil
	x.lk.Unlock()

	for client := range client {
		client.Close()
	}
}

func (x *ClientPool) evict(client DogwoodClientCloser) {
	client.Close()

	x.lk.Lock()
	defer x.lk.Unlock()
	if x.client != nil {
		delete(x.client, client)
	}
}

func (x *ClientPool) Dogwood_null(ctx context.Context) error {
	c, err := x.getClient(ctx)
	if err != nil {
		return err
	}
	err = c.Dogwood_null(ctx)
	if err != nil {
		x.evict(c)
	}
	return err
}

func (x *ClientPool) Dogwood_deliver(ctx context.Context, env proto.DogwoodDeliveryEnvelope) error {
	c, err := x.getClient(ctx)
	if err != nil {
		return err
	}
	span := x.deliverLat.Start()
	err = c.Dogwood_deliver(ctx, env)
	span.Stop()
	if err != nil {
		x.evict(c)
	}
	return err
}
