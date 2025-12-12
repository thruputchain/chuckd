package scheduler

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/coordinator"
	"github.com/groundhogchain/chuckd/test/testsched"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	"github.com/groundhogchain/lib-go/must"
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
	librpc.StartSerial(ctx, x.inside, netutil.MaxMessageSize, opcfg.SchedAddr, proto.SchedulerCoord1_Server{Srv: x.sched})
}

func (x *MockScheduler) ConnectSchedulerToCoordinator(ctx context.Context, opcfg coordinator.OperationalConfig) {
	liblog.Infof("connecting mock scheduler to coordinator on %v at %v", x.inside, opcfg.InAddr)
	coordSchedConn, err := sched.DialCoordSchedulerClient(ctx, x.inside, opcfg.InAddr)
	must.NoError(ctx, err)
	x.sched.SetCoordScheduler(coordSchedConn)
}
