package scheduler

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testsched"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type MockScheduler struct {
	inside libnet.Network
	sched  *testsched.TestScheduler
}

func NewMockScheduler(
	ctx context.Context,
	inside libnet.Network,
	validator proto.ValidatorPrivateInfo,
) *MockScheduler {
	return &MockScheduler{
		inside: inside,
		sched:  testsched.NewTestScheduler(ctx, validator.Public_info, proto.DbTip{Height: 1}),
	}
}

func (x *MockScheduler) StartScheduler(ctx context.Context, opcfg coordinator.OperationalConfig) {
	liblog.Infof("mock scheduler starting on %v at %v", x.inside, opcfg.SchedAddr)
	rpcutil.StartSerial(ctx, x.inside, opcfg.SchedAddr, proto.SchedulerCoord1_Server{Srv: x.sched})
}

func (x *MockScheduler) ConnectSchedulerToCoordinator(ctx context.Context, opcfg coordinator.OperationalConfig) {
	liblog.Infof("connecting mock scheduler to coordinator on %v at %v", x.inside, opcfg.InAddr)
	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, x.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	x.sched.SetCoordScheduler(coordSchedConn)
}
