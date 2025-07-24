package dogwood

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: pick client with smallest number of outstanding connections

type ClientPool struct {
	address     proto.ValidatorID
	time        libtime.Time
	dialer      DogwoodDialer
	maxClients  int
	redialDelay time.Duration
	//
	lk           sync.Mutex
	lastDialFail *time.Time
	clients      map[DogwoodClientCloser]int // client -> number of pending calls
	//
	leaseLat   libstat.LatencyMeter
	deliverLat libstat.LatencyMeter
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
		clients:     map[DogwoodClientCloser]int{},
	}
}

func (x *ClientPool) NumClients() int {
	x.lk.Lock()
	defer x.lk.Unlock()
	if x.clients == nil {
		return 0
	}
	return len(x.clients)
}

var (
	ErrClientPoolClosed  = fmt.Errorf("dogwood client pool: closed")
	ErrClientPoolNotFull = fmt.Errorf("dogwood client pool: not full")
)

func (x *ClientPool) leaseFromPool(ctx context.Context) (client DogwoodClientCloser, err error) {

	x.lk.Lock()
	defer x.lk.Unlock()

	if x.clients == nil {
		return nil, ErrClientPoolClosed
	}

	if len(x.clients) < x.maxClients {
		return nil, ErrClientPoolNotFull
	}

	var minClient DogwoodClientCloser
	var minPending int

	for c, npending := range x.clients {
		if minClient == nil || minPending > npending {
			minClient, minPending = c, npending
		}
	}

	x.clients[minClient] += 1
	return minClient, nil
}

func (x *ClientPool) leaseClient(ctx context.Context) (client DogwoodClientCloser, err error) {

	client, err = x.leaseFromPool(ctx)

	if err == nil {
		return client, nil
	}

	if errors.Is(err, ErrClientPoolClosed) {
		return nil, err
	}

	if !errors.Is(err, ErrClientPoolNotFull) {
		return nil, err
	}

	span := x.leaseLat.Start()

	// pool was not closed and not full, so dialing a new client
	client, err = x.dial(ctx)
	if err != nil {
		return nil, err
	}

	// place the new client in the pool, if it is not closed
	x.lk.Lock()
	closed := x.clients == nil
	if !closed {
		x.clients[client] = 1
	}
	x.lk.Unlock()

	if closed {
		client.Close()
		return nil, ErrClientPoolClosed
	}

	span.Stop()

	return client, nil
}

func (x *ClientPool) releaseClient(ctx context.Context, client DogwoodClientCloser) {
	x.lk.Lock()
	defer x.lk.Unlock()

	if x.clients == nil {
		return
	}

	npending, ok := x.clients[client]
	if ok {
		must.Assert(ctx, npending > 0)
		x.clients[client] -= 1
	}
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
	client := x.clients
	x.clients = nil
	x.lk.Unlock()

	for client := range client {
		client.Close()
	}
}

func (x *ClientPool) evict(client DogwoodClientCloser) {
	client.Close()

	x.lk.Lock()
	defer x.lk.Unlock()
	if x.clients != nil {
		delete(x.clients, client)
	}
}

func (x *ClientPool) Dogwood_deliver(ctx context.Context, env proto.DogwoodDeliveryEnvelope) error {
	c, err := x.leaseClient(ctx)
	if err != nil {
		return err
	}
	defer x.releaseClient(ctx, c)

	span := x.deliverLat.Start()
	err = c.Dogwood_deliver(ctx, env)
	span.Stop()
	if err != nil {
		x.evict(c)
	}
	return err
}

// stats

type ClientPoolStats struct {
	Address    string
	LeaseLat   libstat.LatencyStats
	DeliverLat libstat.LatencyStats
	Clients    []ClientStats
}

type ClientStats struct {
	NumPending int
}

func (x *ClientPool) Stats() ClientPoolStats {

	x.lk.Lock()
	cs := make([]ClientStats, 0, len(x.clients))
	for _, npending := range x.clients {
		cs = append(cs, ClientStats{NumPending: npending})
	}
	x.lk.Unlock()

	return ClientPoolStats{
		Address:    protoutil.VIDString(x.address),
		LeaseLat:   x.leaseLat.Stats(),
		DeliverLat: x.deliverLat.Stats(),
		Clients:    cs,
	}
}
