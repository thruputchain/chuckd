package testcluster

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/test/testdb"
	"github.com/groundhogchain/chuckd/test/testexec"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/chuckd/test/testshard"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestCluster struct {
	net    *testnet.TestNetwork
	db     *testdb.TestDb
	shards []*testshard.TestShard
	execs  []*testexec.TestExec
}

const coordAddress = "coord"

func NewTestCluster(
	ctx context.Context,
	net *testnet.TestNetwork,
) *TestCluster {

	db := testdb.NewTestDb(sdk.Tip{})

	shards := make([]*testshard.TestShard, 2)
	for i := range shards {
		shards[i] = testshard.NewTestShard(ctx, db)
	}

	execs := make([]*testexec.TestExec, 2)
	for i := range execs {
		execs[i] = testexec.NewTestExec(db)
	}

	return &TestCluster{
		net:    net,
		db:     db,
		shards: shards,
		execs:  execs,
	}
}

func (x *TestCluster) ShardAddress(shard int) string {
	return fmt.Sprintf("db_%d", shard)
}

func (x *TestCluster) ExecAddress(shard int) string {
	return fmt.Sprintf("exec_%d", shard)
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

func (x *TestCluster) Start(ctx context.Context, inAddr string) {
	for i, exec := range x.execs {
		rpcutil.Start(ctx, x.net, x.ExecAddress(i), proto.ExecutorCoord1_Server{Srv: exec})
	}
	for i, shard := range x.shards {
		rpcutil.Start(ctx, x.net, x.ShardAddress(i), proto.Shard1_Server{Srv: shard})
	}
	go x.register(ctx, inAddr)
}

func (x *TestCluster) register(ctx context.Context, inAddr string) {

	coord, err := coord.DialCoordinatorClient(ctx, x.net, inAddr)
	must.NoError(ctx, err)
	defer coord.Close()

	for i, _ := range x.shards {
		_, err := coord.Coordinator_intro_shard(ctx, x.ShardCard(i))
		must.NoError(ctx, err)
	}
	for i, _ := range x.execs {
		_, err := coord.Coordinator_intro_exec(ctx, x.ExecCard(i))
		must.NoError(ctx, err)
	}
}
