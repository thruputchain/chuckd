package coordinator_loadgen

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/net/debug"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/loadgen"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/system/systemkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librand"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type CoordinatorLoadgenSystem struct {
	inside      libnet.Network
	outside     libnet.Network
	notify      *coordinator.NotifyReceive
	coordinator *coordinator.System
	scheduler   systemkit.SchedulerSystem
	loadgen     *loadgen.LoadGen
	skip        int
	stopOnce    sync.Once
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
	skip int,
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
		skip:        skip,
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
	liblog.Infof(ctx, "all counterparties are up; starting coordinator benchmark")

	// start scheduler system
	s.scheduler.StartScheduler(ctx, opcfg)

	// start reception fromloadgen
	s.loadgen.Start(ctx, loadgenAddr)

	// start coordinator
	ctl := s.coordinator.Start(ctx, opcfg)

	s.scheduler.ConnectSchedulerToCoordinator(ctx, opcfg)

	// check if genesis exists
	hasGenesis := false
	tipResult, err := s.coordinator.SchedConn.Schedcoord_get_tip(ctx)
	must.NoError(ctx, err)
	if tipResult.Outcome == proto.SchedulerGetTipOK {
		hasGenesis = tipResult.Db_tip().Value.Proposal_header.Block_height >= 0
	}

	// process genesis
	if hasGenesis {
		s.loadgen.WaitGenesis(ctx)

		ctl.SetLevers(batonkit.BatonLevers{PauseNominating: true})
		ctl.SignalGenesisIsReady()

		s.DrainInitTxs(ctx)
	} else {
		genesis := s.loadgen.WaitGenesis(ctx)
		_, err = ctl.LoadGenesisConfiguration(ctx, *genesis)
		must.NoError(ctx, err)

		ctl.SetLevers(batonkit.BatonLevers{PauseNominating: true})
		ctl.SignalGenesisIsReady()

		s.PumpInitTxs(ctx, ctl)
	}

	s.PumpWorkTxs(ctx, ctl, s.skip)
}

func (s *CoordinatorLoadgenSystem) DrainInitTxs(ctx context.Context) {
	i := 0
	for {
		liblog.Infof(ctx, "draining init tx batch %d from loadgen", i)
		txs := s.loadgen.WaitInitTxs(ctx)
		if len(txs) == 0 {
			return
		}
		i += 1
	}
}

func (s *CoordinatorLoadgenSystem) PumpInitTxs(ctx context.Context, ctl coordinator.Control) {
	must.Assert(ctx, ctl.GetLevers().PauseNominating)

	// install nomination gating hook
	uncommitted := map[proto.TransactionHash]bool{}
	ctl.SetHooks(
		batonkit.JustCommitHook(
			// hook pauses nominations iff uncommitted is empty
			func(cb *batonkit.CommittableBlock) batonkit.BatonLevers {
				ns, err := xdrutil.SerializeTransactionHashBatch(cb.AsConsensusBasis().Proposal_txs.Hash)
				must.NoError(ctx, err)
				for _, h := range ns {
					delete(uncommitted, h)
				}
				if len(uncommitted) == 0 {
					liblog.Debugf(ctx, "commit hook: pause nominating")
				} else {
					liblog.Debugf(ctx, "commit hook: continue nominating (%d uncommitted)", len(uncommitted))
				}
				return batonkit.BatonLevers{PauseNominating: len(uncommitted) == 0}
			},
		),
	)
	defer ctl.SetHooks(nil)

	i := 0
	for {
		must.Assert(ctx, ctl.GetLevers().PauseNominating)

		liblog.Infof(ctx, "waiting for init tx batch %d to arrive from loadgen", i)
		txs := s.loadgen.WaitInitTxs(ctx)
		if len(txs) == 0 {
			return
		}

		liblog.Infof(ctx, "submitting init tx batch %d with %d transactions", i, len(txs))
		for _, tx := range txs {
			uncommitted[protoutil.HashSignedTx(tx)] = true
		}
		ctl.SubmitTxs(ctx, txs)

		// nominations resume

		// after resuming, an empty nomination can slip in before submitted txs reach mempool,
		// so we wait for a bit to avoid this
		liblog.Debugf(ctx, "sleeping for 500ms before nominating")
		time.Sleep(500 * time.Millisecond)

		liblog.Debugf(ctx, "resume nominating")
		ctl.ResumeNominating()

		liblog.Infof(ctx, "waiting for init batch %d to commit", i)
		for {
			s.notify.ReceiveCommitted(ctx) // returns after the hook is called
			if len(uncommitted) == 0 {
				// the hook has paused nominations
				must.Assert(ctx, ctl.GetLevers().PauseNominating)
				break
			}
		}

		i += 1
	}
}

type BlockStats struct {
	Height        proto.BlockHeight
	BlockNTxs     int
	BlockDuration time.Duration
	TotalNTxs     int
	TotalDuration time.Duration
}

func (x BlockStats) PrintHogdReport() {
	fmt.Printf("Block duration (s), %.2f, Num txs in block, %d, Overall "+
		"duration (s), %.2f, Overall number of txs, %d\n",
		x.BlockDuration.Seconds(), x.BlockNTxs, x.TotalDuration.Seconds(), x.TotalNTxs)
	fmt.Printf("Block throughput (tx/s), %.2f, Overall throughput (tx/s), %.2f\n",
		float64(x.BlockNTxs)/max(float64(x.BlockDuration.Seconds()), 0.001),
		float64(x.TotalNTxs)/max(float64(x.TotalDuration.Seconds()), 0.001),
	)
}

func (s *CoordinatorLoadgenSystem) PumpWorkTxs(
	ctx context.Context,
	ctl coordinator.Control,
	skip int,
) []BlockStats {
	ctl.ResumeNominating()

	stop := make(chan struct{})
	report := make(chan []BlockStats)

	go func(stop <-chan struct{}, report chan<- []BlockStats) {
		stats := []BlockStats{}
		totalNTxs := 0
		totalStartTime := time.Now()
		j := 0
		for {
			blockStartTime := time.Now()
			select {
			case <-stop:
				report <- stats
				return
			case cb := <-s.notify.ReceiveCommittedChannel():
				totalNTxs += cb.NumTransactions()
				blockStats := BlockStats{
					Height:        cb.BlockHeight(),
					BlockNTxs:     cb.NumTransactions(),
					BlockDuration: time.Since(blockStartTime),
					TotalNTxs:     totalNTxs,
					TotalDuration: time.Since(totalStartTime),
				}
				if j+1 > skip {
					blockStats.PrintHogdReport()
					stats = append(stats, blockStats)
				}
			}
			j += 1
		}
	}(stop, report)

	var lastBurst time.Time
	i := 0
	for {
		liblog.Infof(ctx, "waiting for work tx batch %d to arrive from loadgen", i)
		txs, ok := s.loadgen.WaitWorkTxs(ctx)
		if !ok {
			close(stop)
			return <-report
		}

		// wait until at least a second since last burst of submission
		if delay := time.Until(lastBurst.Add(time.Second)); delay > 0 {
			time.Sleep(delay)
		}
		lastBurst = time.Now()

		liblog.Infof(ctx, "submitting work tx batch %d with %d transactions", i, xdrutil.TokenSizeOfBatch(txs))
		ctl.SubmitParallelTxs(ctx, txs)

		liblog.Infof(ctx, "waiting for work batch %d to commit", i)

		i += 1
	}
}

func (s *CoordinatorLoadgenSystem) Close() {
	s.Stop()
}

func (s *CoordinatorLoadgenSystem) Stop() {
	s.stopOnce.Do(func() {
		s.loadgen.Close()
		s.scheduler.Stop()
		s.coordinator.Stop()
	})
}
