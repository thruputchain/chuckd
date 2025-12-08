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
	"github.com/groundhogchain/chuckd/system/systemkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type CoordinatorLoadgenSystem struct {
	inside      libnet.Network
	outside     libnet.Network
	notify      *coordinator.NotifyReceive
	coordinator *coordinator.System
	scheduler   systemkit.SchedulerSystem
	loadgen     *loadgen.LoadGen
}

func NewCoordinatorLoadgenSystem(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	inside libnet.Network,
	outside libnet.Network,
	config proto.Config,
	loadgenConfig loadgen.Config,
	scheduler systemkit.SchedulerSystem,
) *CoordinatorLoadgenSystem {
	notify := coordinator.NewNotifyReceive()
	cs := coordinator.NewSystem(ctx, notify, librand.NewRandFrom(rand), config, inside, outside)
	lg := loadgen.NewLoadGen(ctx, librand.NewRandFrom(rand), inside, loadgenConfig)
	return &CoordinatorLoadgenSystem{
		inside:      inside,
		outside:     outside,
		notify:      notify,
		coordinator: cs,
		scheduler:   scheduler,
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

	// start scheduler system
	s.scheduler.StartScheduler(ctx, opcfg)

	// record the beginning of the benchmark
	t0 := time.Now()

	// start reception fromloadgen
	s.loadgen.Start(ctx, loadgenAddr)

	// start coordinator
	ctl := s.coordinator.Start(ctx, opcfg)

	// process genesis
	genesis := s.loadgen.WaitGenesis(ctx)
	_, err = ctl.LoadGenesisBlock(ctx, *genesis)
	must.NoError(ctx, err)
	ctl.GenesisIsReady()

	s.scheduler.ConnectSchedulerToCoordinator(ctx, opcfg)

	// pump transactions
	i := 0
	txm := xdrutil.NewXdrMarshaler()
	uncommitted := map[proto.SignedTransactionHash]bool{}
pump:
	for {
		select {
		case <-ctx.Done():
			break pump
		default:
		}

		liblog.Infof("waiting for tx batch %d to arrive from loadgen", i)
		txs := s.loadgen.WaitBlockTxs(ctx)
		if txs == nil {
			break pump
		}

		select {
		case <-ctx.Done():
			break pump
		default:
		}

		liblog.Infof("submitting tx batch %d with %d transactions", i, len(txs))
		for _, tx := range txs {
			uncommitted[protoutil.HashSignedTx(txm, tx)] = true
		}
		ctl.SubmitTxs(ctx, txs)

		liblog.Infof("waiting for batch %d to commit", i)
		for {
			cb := s.notify.ReceiveCommitted(ctx)
			for _, tx := range cb.SignedTransactions() {
				delete(uncommitted, protoutil.HashSignedTx(txm, &tx))
			}
			if len(uncommitted) == 0 {
				break
			}
		}

		i += 1
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
