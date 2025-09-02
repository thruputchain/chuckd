package testshard

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/test/testdb"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestShard struct {
	testdb.TestDb
}

func NewTestShard(ctx context.Context, db testdb.TestDb) *TestShard {
	return &TestShard{TestDb: db}
}

func (s *TestShard) Start(ctx context.Context, net libnet.Network, address string) net.Listener {
	return rpcutil.StartSerial(ctx, net, address, proto.DbCoord1_Server{Srv: s})
}
