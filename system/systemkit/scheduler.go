package systemkit

import (
	"context"

	"github.com/groundhogchain/chuckd/system/coordinator"
)

type SchedulerSystem interface {
	StartScheduler(ctx context.Context, opcfg coordinator.OperationalConfig)
	ConnectSchedulerToCoordinator(ctx context.Context, opcfg coordinator.OperationalConfig)
	Stop()
}
