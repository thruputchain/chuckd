package dogwood

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Dogwood3Harness struct {
	ctx       context.Context
	rand      []*rand.Rand
	vs        []*proto.ValidatorPrivateInfo
	router    []router.Router
	time      libtime.Time
	net       libnet.Network
	discovery []*discovery.TestDiscovery
	dogwood   []*DogwoodService
}

func NewDogwood3Testnet(cfg proto.DogwoodConfig) *Dogwood3Harness {
	return NewDogwood3Harness(cfg, libtime.RealTime{}, testnet.NewTestNetwork(), "alpha", "beta", "gamma")
}

func NewDogwood3TCPNet(cfg proto.DogwoodConfig) *Dogwood3Harness {
	return NewDogwood3Harness(cfg, libtime.RealTime{}, libnet.TCPNet{}, ":55550", ":55551", ":55552")
}

func NewDogwood3Harness(
	cfg proto.DogwoodConfig,
	time_ libtime.Time,
	net libnet.Network,
	a0, a1, a2 string,
) *Dogwood3Harness {
	t := &Dogwood3Harness{
		ctx:  context.Background(),
		time: time_,
		net:  net,
		rand: []*rand.Rand{
			runtime.NewRandFromRoot(),
			runtime.NewRandFromRoot(),
			runtime.NewRandFromRoot(),
		},
	}
	t.vs = []*proto.ValidatorPrivateInfo{
		protoutil.MustGenerateIdentity(t.ctx, t.rand[0], a0),
		protoutil.MustGenerateIdentity(t.ctx, t.rand[1], a1),
		protoutil.MustGenerateIdentity(t.ctx, t.rand[2], a2),
	}
	for i := range t.vs {
		liblog.Infof("dogwood test validator #%d: %s", i, protoutil.VPublicShortString(t.vs[i].Public_info))
	}
	t.discovery = []*discovery.TestDiscovery{
		discovery.NewTestDiscovery(t.ctx, t.vs, 0),
		discovery.NewTestDiscovery(t.ctx, t.vs, 1),
		discovery.NewTestDiscovery(t.ctx, t.vs, 2),
	}
	trees := map[router.TestTreeKey]*router.TestTreeNode{
		// 0 -> 1, 0 -> 2
		{VID: protoutil.VIDOfPublic(t.vs[0].Public_info)}: {
			V: t.vs[0].Public_info,
			Children: []*router.TestTreeNode{
				{V: t.vs[1].Public_info},
				{V: t.vs[2].Public_info},
			},
		},
		// 1 -> 2 -> 0
		{VID: protoutil.VIDOfPublic(t.vs[1].Public_info)}: {
			V: t.vs[1].Public_info,
			Children: []*router.TestTreeNode{
				{
					V: t.vs[2].Public_info,
					Children: []*router.TestTreeNode{
						{V: t.vs[0].Public_info},
					},
				},
			},
		},
		// 2 -> 0 -> 1
		{VID: protoutil.VIDOfPublic(t.vs[2].Public_info)}: {
			V: t.vs[2].Public_info,
			Children: []*router.TestTreeNode{
				{
					V: t.vs[0].Public_info,
					Children: []*router.TestTreeNode{
						{V: t.vs[1].Public_info},
					},
				},
			},
		},
	}
	t.router = []router.Router{
		router.NewTestRouter(t.vs[0].Public_info, trees),
		router.NewTestRouter(t.vs[1].Public_info, trees),
		router.NewTestRouter(t.vs[2].Public_info, trees),
	}
	t.dogwood = []*DogwoodService{
		NewDogwoodService(t.ctx, cfg, t.time, t.rand[0], t.net, t.discovery[0], t.router[0]),
		NewDogwoodService(t.ctx, cfg, t.time, t.rand[1], t.net, t.discovery[1], t.router[1]),
		NewDogwoodService(t.ctx, cfg, t.time, t.rand[2], t.net, t.discovery[2], t.router[2]),
	}
	return t
}

func (x *Dogwood3Harness) VID(i int) proto.ValidatorID {
	return protoutil.VIDOfPublic(x.vs[i].Public_info)
}

func (x *Dogwood3Harness) Time() libtime.Time {
	return x.time
}

func (x *Dogwood3Harness) Dogwood(i int) *DogwoodService {
	return x.dogwood[i]
}

func (x *Dogwood3Harness) Start(ctx context.Context) {
	x.dogwood[0].Start(ctx, x.vs[0].Public_info.Address)
	x.dogwood[1].Start(ctx, x.vs[1].Public_info.Address)
	x.dogwood[2].Start(ctx, x.vs[2].Public_info.Address)
}
