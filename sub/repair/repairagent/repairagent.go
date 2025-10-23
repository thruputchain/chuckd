package repairagent

import (
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairAgent struct {
	backoffCfg netutil.BackoffConfig
	net        libnet.AddressDialer
	signatory  protoutil.Signatory
	txM        xdrutil.XdrMarshaler
	cvM        xdrutil.XdrMarshaler
}

func NewRepairAgent(
	cfg proto.RepairConfig,
	net libnet.AddressDialer,
	signatory protoutil.Signatory,
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
		signatory: signatory,
		txM:       xdrutil.NewXdrMarshaler(),
		cvM:       xdrutil.NewXdrMarshaler(),
	}
}
