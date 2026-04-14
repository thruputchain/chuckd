package scheduler

import (
	"context"

	"github.com/groundhogchain/chuckd/system/coordinator"
)

type LiveScheduler struct{}

func NewLiveScheduler() *LiveScheduler {
	return &LiveScheduler{}
}

func (x *LiveScheduler) StartScheduler(ctx context.Context, opcfg coordinator.OperationalConfig) {
	// live scheduler is started by operator
}

func (x *LiveScheduler) ConnectSchedulerToCoordinator(ctx context.Context, opcfg coordinator.OperationalConfig) {
	// live scheduler connects to coordinator on its own
}

func (x *LiveScheduler) Stop() {}
