package testcluster

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/test/testdb"
	"github.com/groundhogchain/chuckd/test/testexec"
	"github.com/groundhogchain/chuckd/test/testshard"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestCluster struct {
	net       libnet.Network
	cfg       TestClusterConfig
	db        testdb.TestDb
	gen       *testutil.TestGen
	submitter batonapi.Submitter
	shards    []*testshard.TestShard
	execs     []*testexec.TestExec
}

type TestClusterConfig struct {
	TPS          int64 // txs per sec, submitted by the test cluster driver to the submitter; can be 0
	NumDb        int
	NumExec      int
	DbBasePort   int
	ExecBasePort int
}

func NewTestCluster(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	net libnet.Network,
	submitter batonapi.Submitter,
	cfg TestClusterConfig,
	speed bool,
) *TestCluster {

	db := testdb.NewTestDbConsistent(batonapi.Tip{})

	shards := make([]*testshard.TestShard, cfg.NumDb)
	for i := range shards {
		shards[i] = testshard.NewTestShard(ctx, db)
	}

	execs := make([]*testexec.TestExec, cfg.NumExec)
	for i := range execs {
		if speed {
			// give execs individual dbs to prevent cache bouncing between processor cores
			execs[i] = testexec.NewTestExec(testdb.NewTestDbNoop())
		} else {
			execs[i] = testexec.NewTestExec(db)
		}
	}

	userPriv, err := protoutil.GenerateIdentity(rand, "")
	must.NoError(ctx, err)

	return &TestCluster{
		net:       net,
		cfg:       cfg,
		db:        db,
		gen:       testutil.NewTestGen(ctx, time, rand, userPriv.Secret_key),
		submitter: submitter,
		shards:    shards,
		execs:     execs,
	}
}

func (x *TestCluster) ShardAddress(shard int) string {
	return fmt.Sprintf("localhost:%d", x.cfg.DbBasePort+shard)
}

func (x *TestCluster) ExecAddress(shard int) string {
	return fmt.Sprintf("localhost:%d", x.cfg.ExecBasePort+shard)
}

func (x *TestCluster) ShardCard(shard int) proto.ShardCard {
	return proto.ShardCard{
		Upid:         xdrutil.Uint64UPID(5 * uint64(shard)),
		Block_height: x.db.GetTip().Height,
		Shard_count:  uint32(len(x.shards)),
		Address:      x.ShardAddress(shard),
	}
}

func (x *TestCluster) ExecCard(shard int) proto.ExecCard {
	return proto.ExecCard{
		Protocol: "",
		Upid:     xdrutil.Uint64UPID(23 * uint64(shard)),
		Address:  x.ExecAddress(shard),
	}
}

func (x *TestCluster) Start(ctx context.Context, coordAddr string) {
	liblog.Infof("testcluster starting")
	for i, exec := range x.execs {
		rpcutil.StartSerial(ctx, x.net, x.ExecAddress(i), proto.ExecutorCoord1_Server{Srv: exec})
	}
	for i, shard := range x.shards {
		rpcutil.StartSerial(ctx, x.net, x.ShardAddress(i), proto.Shard1_Server{Srv: shard}, proto.DbCoord1_Server{Srv: shard})
	}
	go x.register(ctx, coordAddr)
	go x.pumpTxs(ctx)
}

func (x *TestCluster) register(ctx context.Context, coordAddr string) {
	liblog.Infof("testcluster registering with coordinator %v", coordAddr)

	coord, err := coord.DialCoordinatorClient(ctx, x.net, coordAddr)
	must.NoError(ctx, err)
	defer coord.Close()

	liblog.Infof("testcluster introducing %d shards", len(x.shards))
	for i := range x.shards {
		_, err := coord.Coordinator_intro_shard(ctx, x.ShardCard(i))
		must.NoError(ctx, err)
	}
	liblog.Infof("testcluster introducing %d execs", len(x.execs))
	for i := range x.execs {
		_, err := coord.Coordinator_intro_exec(ctx, x.ExecCard(i))
		must.NoError(ctx, err)
	}
}

func (x *TestCluster) pumpTxs(ctx context.Context) {
	liblog.Infof("testcluster pumping txs")
	for {
		time.Sleep(time.Millisecond * 500) // 1/2 of second resolution
		for i := 0; i < int(x.cfg.TPS)/2; i++ {
			x.submitter.SubmitTxs(ctx, x.gen.GenSignedTx(ctx))
		}
	}
}
