package repair

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/must"
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
	<-x.ready // wait for x.conn != nil

	// XXX: verify env.Missing_transactions.Block_hash is our nomination, and
	// XXX: requested txs are part of our nomination

	found, err := x.conn.Schedcoord_lookup_transactions(ctx, env.Missing_transactions.Transaction_hashes)
	must.NoError(ctx, err)
	must.Assert(ctx, len(found) == len(env.Missing_transactions.Transaction_hashes))

	txs := make([]proto.SignedTransaction, len(env.Missing_transactions.Transaction_hashes))
	for i, otx := range found {
		// we must have all transactions that we have nominated, or else
		// the request is invalid
		if otx == nil {
			return proto.RepairedTransactions{}, nil
		}
		txs[i] = *otx
	}

	return proto.RepairedTransactions{
		Transactions: txs,
	}, nil
}
