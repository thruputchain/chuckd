package repairagent

import (
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librand"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairAgent struct {
	backoffCfg netutil.BackoffConfig
	net        libnet.Dialer
	rand       *librand.SyncRand
	signatory  protoutil.Signatory
	router     router.Router
	cvpool     *cvpool.ConsensusValuePool
	txM        libxdr.XdrMarshaler
}

func NewRepairAgent(
	cfg proto.RepairConfig,
	net libnet.Dialer,
	rand *rand.Rand,
	signatory protoutil.Signatory,
	router router.Router,
	cvpool *cvpool.ConsensusValuePool,
) *RepairAgent {
	return &RepairAgent{
		backoffCfg: netutil.BackoffConfig{
			BaseDelay:  time.Duration(cfg.Base_delay_msec) * time.Millisecond,
			MaxDelay:   time.Duration(cfg.Max_delay_msec) * time.Millisecond,
			Multiplier: float64(cfg.Multiplier_nom) / float64(cfg.Multiplier_denom),
			MaxRetries: 0,
			JitterType: netutil.EqualJitter,
		},
		net:       net,
		rand:      librand.NewSyncRandWrapping(rand),
		signatory: signatory,
		router:    router,
		cvpool:    cvpool,
		txM:       libxdr.NewXdrMarshaler(),
	}
}
