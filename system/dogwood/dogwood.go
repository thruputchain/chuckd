package dogwood

import (
	"context"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/router"
	proto "github.com/groundhogchain/protocol-go"
)

type System struct {
	cfg       proto.DogwoodStandaloneConfig
	discovery discovery.Discovery
	dogwood   *dogwood.DogwoodService
}

func NewSystem(
	ctx context.Context,
	rand *rand.Rand,
	net libnet.Network,
	cfg proto.DogwoodStandaloneConfig,
) *System {

	discovery := discovery.NewStaticDiscovery(
		cfg.Validator_config.Private_info,
		cfg.Validator_config.Known_validators,
	)

	router := router.NewStaticUniformRouter(
		ctx,
		cfg.Validator_config.Private_info.Public_info,
		cfg.Validator_config.Known_validators,
		int(cfg.Router_config.Tree_fan_out),
	)

	dogwood := dogwood.NewDogwoodService(
		ctx,
		cfg.Dogwood_config,
		libtime.RealTime{},
		rand,
		net,
		net,
		discovery,
		router,
	)
	return &System{
		cfg:       cfg,
		discovery: discovery,
		dogwood:   dogwood,
	}
}

func (sys *System) Start(ctx context.Context, extAddr string, optInAddr string) {
	sys.dogwood.Start(ctx, extAddr, optInAddr, 3*time.Second)
	sys.dogwood.WaitReady(ctx)
}
