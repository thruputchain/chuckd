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

	x := &DogwoodService{}
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
		int(cfg.Endpoint_queue_length), int(cfg.Endpoint_max_pending), int(cfg.Endpoint_num_clients),
		endpointIdleTTL, endpointRedialDelay,
	)

	x.streamReceiver = NewStreamReceiver(
		ctx, discovery, x.nav, x.caller,
		int(cfg.Stream_receiver_queue_length), int(cfg.Stream_receiver_worker_count),
	)

	x.streamSender = NewStreamSender(
		ctx, librand.NewFromRand(rand), discovery, x.nav, x.caller,
		int(cfg.Stream_sender_worker_count),
	)

	x.xReceiver = NewTransmissionReceiver(
		ctx, *cfg.Transmission_config, time_, x.nav, x.streamReceiver,
		int(cfg.Transmission_receiver_worker_count),
	)

	x.xSender = NewTransmissionSender(
		ctx, *cfg.Transmission_config, x.nav, x.streamSender, librand.NewFromRand(rand),
		int(cfg.Transmission_sender_worker_count),
	)

	x.mplex = NewMultiplex(ctx, x.router, x.xSender, x.xReceiver, x.nav)

	return x
}

func (x *DogwoodService) Start(ctx context.Context, address string) {
	// only start systems constructed by dogwood service
	x.dialer.Start(ctx)         // no dependency
	x.caller.Start(ctx)         // depends on dialer
	x.streamReceiver.Start(ctx) // depends on discovery, navigator, and caller
	x.streamSender.Start(ctx)   // depends on discovery, navigator, and caller
	x.xReceiver.Start(ctx)      // depends on navigator, and stream receiver
	x.xSender.Start(ctx)        // depends on navigator, and stream sender
	x.mplex.Start(ctx)          // depends on navigator, transmission receiver, and transmission sender
	x.nav.Start(ctx, x.mplex)   // depends on router and multiplex

	rpcutil.Start(ctx, x.net, address, proto.Dogwood1_Server{Srv: x})
}
