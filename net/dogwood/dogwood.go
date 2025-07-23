package dogwood

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type DogwoodService struct {
	time           libtime.Time
	net            libnet.Network
	router         router.DogwoodRouter
	dialer         DogwoodDialer
	nav            *Navigator
	caller         Caller
	streamReceiver *StreamReceiver
	streamSender   *StreamSender
	xReceiver      *TransmissionReceiver
	xSender        *TransmissionSender
	mplex          *Multiplex
	//
	ready chan struct{}
}

func NewDogwoodService(
	ctx context.Context,
	cfg proto.DogwoodConfig,
	time_ libtime.Time,
	rand *rand.Rand,
	net libnet.Network,
	discovery discovery.Discovery,
	router router.DogwoodRouter,
) *DogwoodService {

	x := &DogwoodService{
		ready: make(chan struct{}),
	}
	x.time = time_
	x.net = net
	x.router = router
	x.dialer = NewDogwoodClientDialer(net, router)

	routeTTLive := time.Millisecond * time.Duration(cfg.Route_time_to_live_msec)
	routeTTUse := time.Millisecond * time.Duration(cfg.Route_time_to_use_msec)

	x.nav = NewNavigator(
		ctx, time_, librand.NewFromRand(rand), x.router,
		routeTTLive, routeTTUse, int(cfg.Route_count),
	)

	endpointIdleTTL := time.Millisecond * time.Duration(cfg.Endpoint_idle_ttl_msec)
	endpointRedialDelay := time.Millisecond * time.Duration(cfg.Endpoint_redial_delay_msec)

	x.caller = NewCaller(
		ctx, time_, x.dialer,
		int(cfg.Endpoint_max_pending), int(cfg.Endpoint_num_clients),
		endpointIdleTTL, endpointRedialDelay,
	)

	x.streamReceiver = NewStreamReceiver(
		ctx, time_, discovery, x.nav, x.caller,
		int(cfg.Stream_receiver_worker_count),
	)

	x.streamSender = NewStreamSender(
		ctx, librand.NewFromRand(rand), discovery, x.nav, x.caller,
		int(cfg.Stream_sender_worker_count),
	)

	x.xReceiver = NewTransmissionReceiver(
		ctx, *cfg.Transmission_config, time_, x.nav, x.streamReceiver,
		int(cfg.Transmission_receiver_decode_worker_count),
	)

	x.xSender = NewTransmissionSender(
		ctx, *cfg.Transmission_config, x.nav, x.streamSender, librand.NewFromRand(rand),
		int(cfg.Transmission_sender_worker_count),
	)

	x.mplex = NewMultiplex(ctx, x.router, x.xSender, x.xReceiver, x.nav)

	return x
}

func (x *DogwoodService) Start(ctx context.Context, address string, grace time.Duration) {
	go func() {
		defer close(x.ready)

		x.dialer.Start(ctx)         // nop
		x.caller.Start(ctx)         // expires idle connections
		x.streamReceiver.Start(ctx) // nop
		x.streamSender.Start(ctx)   // nop
		x.xReceiver.Start(ctx)      // expires partial receptions
		x.xSender.Start(ctx)        // nop
		x.mplex.Start(ctx)          // nop

		// enable peers to connect and call us
		// navigation and delivery will not function until navigator is started later
		rpcutil.Start(ctx, x.net, address, proto.Dogwood1_Server{Srv: x})

		// allow a grace period for peers to start serving
		x.time.Sleep(grace)

		x.nav.Start(ctx, x.mplex) // expire, refill, disseminate routes periodically
		x.nav.WaitReady(ctx)      // wait until the first set of routes are disseminated
	}()
}

func (x *DogwoodService) WaitReady(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-x.ready:
		return nil
	}
}
