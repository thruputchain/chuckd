package scheduler

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testsched"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type MockScheduler struct {
	inside libnet.Network
	sched  *testsched.TestScheduler
	l      net.Listener
	coord  sched.CoordSchedulerConn
}

func NewMockScheduler(
	ctx context.Context,
	inside libnet.Network,
	validator proto.ValidatorPrivateInfo,
	dbDir string,
) *MockScheduler {
	return &MockScheduler{
		inside: inside,
		sched:  testsched.NewTestScheduler(ctx, validator.Public_info, proto.DbTip{
			Value: proto.ConsensusValue{
				Proposal_header: proto.ProposalHeader{Block_height: 0},
			},
		}, dbDir),
	}
}

func (x *MockScheduler) StartScheduler(ctx context.Context, opcfg coordinator.OperationalConfig) {
	liblog.Infof(ctx, "mock scheduler starting on %v at %v", x.inside, opcfg.SchedAddr)
	x.l = librpc.StartSerial(ctx, x.inside, netutil.MaxMessageSize, rpcutil.BaseDriverRegime, opcfg.SchedAddr, proto.SchedulerCoord1_Server{Srv: x.sched})
}

func (x *MockScheduler) ConnectSchedulerToCoordinator(ctx context.Context, opcfg coordinator.OperationalConfig) {
	liblog.Infof(ctx, "connecting mock scheduler to coordinator on %v at %v", x.inside, opcfg.InAddr)
	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, x.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	x.coord = coordSchedConn
	x.sched.SetCoordScheduler(coordSchedConn)
}

func (x *MockScheduler) Stop() {
	if x.coord != nil {
		x.coord.Close()
		x.coord = nil
	}
	if x.l != nil {
		_ = x.l.Close()
		x.l = nil
	}
}
