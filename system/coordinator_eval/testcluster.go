package coordinator_eval

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/kit"
	"github.com/groundhogchain/chuckd/test/testsched"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestCluster struct {
	signer    protoutil.Signatory
	net       libnet.Network
	cfg       TestClusterConfig
	sched     *testsched.TestScheduler
	gen       *testutil.TestGen
	submitter kit.SignedTxSubmitter
}

type TestClusterConfig struct {
	TxsPerSec int64 // txs per sec, submitted by the test cluster driver to the submitter; can be 0
	SchedPort int
}

func NewTestCluster(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	net libnet.Network,
	validator proto.ValidatorPrivateInfo,
	submitter kit.SignedTxSubmitter,
	cfg TestClusterConfig,
	speed bool,
) *TestCluster {

	sched := testsched.NewTestScheduler(ctx, validator.Public_info, proto.DbTip{})

	userPriv, err := protoutil.GenerateIdentity(rand, "")
	must.NoError(ctx, err)

	return &TestCluster{
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

func (x *TestCluster) SchedAddress() string {
	return fmt.Sprintf("localhost:%d", x.cfg.SchedPort)
}

func (x *TestCluster) Start(ctx context.Context, coordAddr string) {

	liblog.Infof("testcluster starting")
	rpcutil.StartSerial(ctx, x.net, x.SchedAddress(), proto.SchedulerCoord1_Server{Srv: x.sched})

	go x.pumpTxs(ctx)
}

func (x *TestCluster) pumpTxs(ctx context.Context) {
	liblog.Infof("testcluster pumping txs")
	for {
		time.Sleep(time.Millisecond * 500) // 1/2 of second resolution
		for i := 0; i < int(x.cfg.TxsPerSec)/2; i++ {
			x.submitter.SubmitTxs(ctx, x.gen.GenSignedTx(ctx))
		}
	}
}
