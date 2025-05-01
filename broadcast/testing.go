package broadcast

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/test"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type Broadcast3TestNet struct {
	Ctx    context.Context
	Rand   []*rand.Rand
	V      []*proto.ValidatorPrivateInfo
	Router router.Router
	Net    *test.TestNetwork
	B      []*BroadcastService
}

func NewBroadcast3TestNet(cfg proto.BroadcastConfig) *Broadcast3TestNet {

	t := &Broadcast3TestNet{
		Ctx: context.Background(),
		Rand: []*rand.Rand{
			runtime.NewRandFromRoot(),
			runtime.NewRandFromRoot(),
			runtime.NewRandFromRoot(),
		},
	}
	t.V = []*proto.ValidatorPrivateInfo{
		protoutil.MustGenerateIdentity(t.Ctx, t.Rand[0], "alpha"),
		protoutil.MustGenerateIdentity(t.Ctx, t.Rand[1], "beta"),
		protoutil.MustGenerateIdentity(t.Ctx, t.Rand[2], "gamma"),
	}
	for i := range t.V {
		log.Infof("test validator #%d: %s\n", i, protoutil.ValidatorPublicShortString(t.V[i].Public_info))
	}
	t.Router = &router.TestRouter{
		Trees: map[router.TestTreeKey]*router.TestTreeNode{
			// 0 -> 1, 0 -> 2
			{VID: protoutil.ValidatorIDFromPublic(t.V[0].Public_info)}: {
				V: t.V[0].Public_info,
				Children: []*router.TestTreeNode{
					{V: t.V[1].Public_info},
					{V: t.V[2].Public_info},
				},
			},
			// 1 -> 2 -> 0
			{VID: protoutil.ValidatorIDFromPublic(t.V[1].Public_info)}: {
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
			{VID: protoutil.ValidatorIDFromPublic(t.V[2].Public_info)}: {
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
		},
	}
	t.Net = test.NewTestNetwork()
	t.B = []*BroadcastService{
		NewBroadcastService(t.Ctx, cfg, t.Net, t.Rand[0], *t.V[0], t.Router, AcceptEverything{}),
		NewBroadcastService(t.Ctx, cfg, t.Net, t.Rand[1], *t.V[1], t.Router, AcceptEverything{}),
		NewBroadcastService(t.Ctx, cfg, t.Net, t.Rand[2], *t.V[2], t.Router, AcceptEverything{}),
	}
	return t
}

func (t *Broadcast3TestNet) Start() {
	t.B[0].Start(t.Ctx, t.V[0].Public_info.Address)
	t.B[1].Start(t.Ctx, t.V[1].Public_info.Address)
	t.B[2].Start(t.Ctx, t.V[2].Public_info.Address)
}
