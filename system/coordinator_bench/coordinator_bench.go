package coordinator_bench

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/sub/kit"
	"github.com/groundhogchain/chuckd/test/testsched"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type CoordinatorBenchCluster struct {
	signer    protoutil.Signatory
	net       libnet.Network
	cfg       CoordinatorBenchConfig
	sched     *testsched.TestScheduler
	gen       *testutil.TestGen
	submitter kit.SignedTxSubmitter
	l         net.Listener
	stopPump  chan struct{}
	stopOnce  sync.Once
}

type CoordinatorBenchConfig struct {
	TxsPerSec int64 // txs per sec, submitted by the test cluster driver to the submitter; can be 0
	SchedPort int
}

func NewCoordinatorBenchCluster(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	net libnet.Network,
	validator proto.ValidatorPrivateInfo,
	submitter kit.SignedTxSubmitter,
	cfg CoordinatorBenchConfig,
	speed bool,
	dbDir string,
) *CoordinatorBenchCluster {

	sched := testsched.NewTestScheduler(
		ctx,
		validator.Public_info,
		proto.DbTip{
			Value: proto.ConsensusValue{
				Proposal_header: proto.ProposalHeader{Block_height: 0},
			},
		},
		dbDir,
	)

	userPriv, err := protoutil.GenerateIdentity(rand, "")
	must.NoError(ctx, err)

	return &CoordinatorBenchCluster{
		net:   net,
		cfg:   cfg,
		sched: sched,
		gen: testutil.NewTestGen(
			ctx,
			time,
			rand,
			validator,
			userPriv.Secret_key,
		),
		submitter: submitter,
		stopPump:  make(chan struct{}),
	}
}

func (x *CoordinatorBenchCluster) SchedAddress() string {
	return fmt.Sprintf("localhost:%d", x.cfg.SchedPort)
}

func (x *CoordinatorBenchCluster) StartScheduler(ctx context.Context) {
	liblog.Infof(ctx, "starting mock scheduler")
	x.l = librpc.StartSerial(ctx, x.net, netutil.MaxMessageSize, rpcutil.BaseDriverRegime, x.SchedAddress(), proto.SchedulerCoord1_Server{Srv: x.sched})
}

func (x *CoordinatorBenchCluster) StartPump(ctx context.Context) {
	liblog.Infof(ctx, "starting txs pump")
	go x.pumpTxs(ctx)
}

func (x *CoordinatorBenchCluster) pumpTxs(ctx context.Context) {
	liblog.Infof(ctx, "testcluster pumping txs")
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-x.stopPump:
			return
		case <-t.C:
			for i := 0; i < int(x.cfg.TxsPerSec)/2; i++ {
				x.submitter.SubmitTxs(ctx, x.gen.GenSignedTx(ctx))
			}
		}
	}
}

func (x *CoordinatorBenchCluster) Stop() {
	x.stopOnce.Do(func() {
		close(x.stopPump)
	})
	if x.l != nil {
		_ = x.l.Close()
		x.l = nil
	}
}
