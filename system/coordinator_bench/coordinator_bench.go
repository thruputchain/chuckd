package coordinator_bench

import (
	"context"
	"fmt"
	"math/rand"
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
) *CoordinatorBenchCluster {

	sched := testsched.NewTestScheduler(
		ctx,
		validator.Public_info,
		proto.DbTip{Height: 0},
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
	}
}

func (x *CoordinatorBenchCluster) SchedAddress() string {
	return fmt.Sprintf("localhost:%d", x.cfg.SchedPort)
}

func (x *CoordinatorBenchCluster) StartScheduler(ctx context.Context) {
	liblog.Infof("starting mock scheduler")
	librpc.StartSerial(ctx, x.net, netutil.MaxMessageSize, rpcutil.BaseDriverRegime, x.SchedAddress(), proto.SchedulerCoord1_Server{Srv: x.sched})
}

func (x *CoordinatorBenchCluster) StartPump(ctx context.Context) {
	liblog.Infof("starting txs pump")
	go x.pumpTxs(ctx)
}

func (x *CoordinatorBenchCluster) pumpTxs(ctx context.Context) {
	liblog.Infof("testcluster pumping txs")
	for {
		time.Sleep(time.Millisecond * 500) // 1/2 of second resolution
		for i := 0; i < int(x.cfg.TxsPerSec)/2; i++ {
			x.submitter.SubmitTxs(ctx, x.gen.GenSignedTx(ctx))
		}
	}
}
