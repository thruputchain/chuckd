package coordinator_loadgen

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/debug"
	"github.com/groundhogchain/chuckd/sub/loadgen"
	"github.com/groundhogchain/chuckd/system/coordinator"
	proto "github.com/groundhogchain/protocol-go"
)

type CoordinatorLoadgenSystem struct {
	inside      libnet.Network
	outside     libnet.Network
	notify      *coordinator.NotifyReceive
	coordinator *coordinator.System
	loadgen     *loadgen.LoadGen
}

func NewCoordinatorLoadgenSystem(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	inside libnet.Network,
	outside libnet.Network,
	cfg proto.Config,
	lgCfg loadgen.Config,
) *CoordinatorLoadgenSystem {
	notify := coordinator.NewNotifyReceive()
	cs := coordinator.NewSystem(ctx, notify, librand.NewRandFrom(rand), cfg, inside, outside)
	lg := loadgen.NewLoadGen(ctx, librand.NewRandFrom(rand), inside, lgCfg)
	return &CoordinatorLoadgenSystem{
		inside:      inside,
		outside:     outside,
		notify:      notify,
		coordinator: cs,
		loadgen:     lg,
	}
}

func (s *CoordinatorLoadgenSystem) Start(
	ctx context.Context,
	opcfg coordinator.OperationalConfig,
	loadgenAddr string,
	ready chan<- struct{},
) {
	defer close(ready)

	// start debug service and wait until all counterparties are ready
	counterAddr := make([]string, len(s.coordinator.Config().Validator_config.Known_validators))
	colonPort := fmt.Sprintf(":%d", opcfg.DebugPort)
	for i, v := range s.coordinator.Config().Validator_config.Known_validators {
		host, _, err := libnet.StripPort(v.Address)
		must.NoError(ctx, err)
		counterAddr[i] = host + colonPort
	}
	err := debug.StartAndWaitCounterpartiesReady(ctx, s.outside, colonPort, counterAddr)
	must.NoError(ctx, err)
	liblog.Infof("all counterparties are up; starting coordinator benchmark")

	// record the beginning of the benchmark
	t0 := time.Now()

	// start
	s.loadgen.Start(ctx, loadgenAddr)
	ctl := s.coordinator.Start(ctx, opcfg)

	// pump genesis
	genesis := s.loadgen.WaitGenesis(ctx)
	_, err = ctl.LoadGenesisBlock(ctx, *genesis)
	must.NoError(ctx, err)
	ctl.GenesisIsReady()

	// pump transactions
pump:
	for {
		select {
		case <-ctx.Done():
			break pump
		default:
		}

		txs := s.loadgen.WaitBlockTxs(ctx)
		if txs == nil {
			break pump
		}

		select {
		case <-ctx.Done():
			break pump
		default:
		}

		ctl.SubmitTxs(ctx, txs)

		// wait until the block is committed
		s.notify.ReceiveCommitted(ctx)
	}

	// print results
	benchDuration := time.Since(t0)
	fmt.Println(
		float64(benchDuration.Milliseconds()) / float64(s.loadgen.Cfg.NumBlocks), // milliseconds/commit
	)
}

func (s *CoordinatorLoadgenSystem) Close() {
	s.coordinator.Close()
	s.loadgen.Close()
}
