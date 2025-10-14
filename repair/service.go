package repair

import (
	"context"

	"github.com/groundhogchain/chuckd/sched"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairService struct {
	conn  sched.SchedulerCoordConn
	ready chan struct{}
}

func NewRepairService() *RepairService {
	return &RepairService{
		ready: make(chan struct{}),
	}
}

func (x *RepairService) SetSchedConn(conn sched.SchedulerCoordConn) {
	x.conn = conn
	close(x.ready)
}

func (x *RepairService) Repair_null(ctx context.Context) error {
	return nil
}

// XXX: verify block_hash is our nomination; verify requester is worth responding to, and their signature
func (x *RepairService) Repair_missing_txs(
	ctx context.Context,
	env proto.MissingTransactionsEnvelope,
) (proto.RepairedTransactions, error) {
	<-x.ready

	// XXX: loookup transactions from mempool
	panic("XXX")

	// return proto.RepairedTransactions{
	// 	Transactions: XXX,
	// }, nil
}
