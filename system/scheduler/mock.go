package scheduler

import (
	"context"
	"net"

	"github.com/thruputchain/chuckd/sub/sched"
	"github.com/thruputchain/chuckd/system/coordinator"
	"github.com/thruputchain/chuckd/test/testsched"
	"github.com/thruputchain/chuckd/util/netutil"
	"github.com/thruputchain/chuckd/util/rpcutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/librpc"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
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
		sched:  testsched.NewTestScheduler(ctx, validator.Public_info, proto.DbTip{}, dbDir),
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
