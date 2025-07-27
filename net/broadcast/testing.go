package broadcast

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Broadcast3Harness struct {
	Ctx    context.Context
	Rand   []*rand.Rand
	V      []*proto.ValidatorPrivateInfo
	Router []router.Router
	Net    libnet.Network
	B      []*BroadcastService
}

func NewBroadcast3Testnet(cfg proto.BroadcastConfig) *Broadcast3Harness {
	return NewBroadcast3Harness(cfg, testnet.NewTestNetwork(), "alpha", "beta", "gamma")
}

func NewBroadcast3TCP(cfg proto.BroadcastConfig) *Broadcast3Harness {
	return NewBroadcast3Harness(cfg, libnet.TCPNet{}, ":55550", ":55551", ":55552")
}

func NewBroadcast3Harness(cfg proto.BroadcastConfig, n libnet.Network, a0, a1, a2 string) *Broadcast3Harness {
	t := &Broadcast3Harness{
		Ctx: context.Background(),
		Rand: []*rand.Rand{
			runtime.NewRandFromRoot(),
			runtime.NewRandFromRoot(),
			runtime.NewRandFromRoot(),
		},
	}
	t.V = []*proto.ValidatorPrivateInfo{
		protoutil.MustGenerateIdentity(t.Ctx, t.Rand[0], a0),
		protoutil.MustGenerateIdentity(t.Ctx, t.Rand[1], a1),
		protoutil.MustGenerateIdentity(t.Ctx, t.Rand[2], a2),
	}
	for i := range t.V {
		liblog.Infof("test validator #%d: %s", i, protoutil.VPublicShortString(t.V[i].Public_info))
	}
	trees := map[router.TestTreeKey]*router.TestTreeNode{
		// 0 -> 1, 0 -> 2
		{VID: protoutil.PublicVID(t.V[0].Public_info)}: {
			V: t.V[0].Public_info,
			Children: []*router.TestTreeNode{
				{V: t.V[1].Public_info},
				{V: t.V[2].Public_info},
			},
		},
		// 1 -> 2 -> 0
		{VID: protoutil.PublicVID(t.V[1].Public_info)}: {
			V: t.V[1].Public_info,
			Children: []*router.TestTreeNode{
				{
					V: t.V[2].Public_info,
					Children: []*router.TestTreeNode{
						{V: t.V[0].Public_info},
					},
				},
			},
		},
		// 2 -> 0 -> 1
		{VID: protoutil.PublicVID(t.V[2].Public_info)}: {
			V: t.V[2].Public_info,
			Children: []*router.TestTreeNode{
				{
					V: t.V[0].Public_info,
					Children: []*router.TestTreeNode{
						{V: t.V[1].Public_info},
					},
				},
			},
		},
	}
	t.Router = []router.Router{
		router.NewTestRouter(t.V[0].Public_info, trees),
		router.NewTestRouter(t.V[1].Public_info, trees),
		router.NewTestRouter(t.V[2].Public_info, trees),
	}
	t.Net = n
	t.B = []*BroadcastService{
		NewBroadcastService(t.Ctx, cfg, t.Net, t.Rand[0], *t.V[0], t.Router[0], AcceptEverything{}),
		NewBroadcastService(t.Ctx, cfg, t.Net, t.Rand[1], *t.V[1], t.Router[1], AcceptEverything{}),
		NewBroadcastService(t.Ctx, cfg, t.Net, t.Rand[2], *t.V[2], t.Router[2], AcceptEverything{}),
	}
	return t
}

func (t *Broadcast3Harness) Start() {
	t.B[0].Start(t.Ctx, t.V[0].Public_info.Address)
	t.B[1].Start(t.Ctx, t.V[1].Public_info.Address)
	t.B[2].Start(t.Ctx, t.V[2].Public_info.Address)
}
