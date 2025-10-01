package repair

import (
	"context"

	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairService struct {
	submitted *mempool.SubmittedMempool
}

func NewRepairService(_ context.Context, submitted *mempool.SubmittedMempool) *RepairService {
	return &RepairService{
		submitted: submitted,
	}
}

func (x *RepairService) Repair_null(ctx context.Context) error {
	return nil
}

// XXX: verify block_hash is our nomination; verify requester is worth responding to, and their signature
func (x *RepairService) Repair_missing_txs(
	ctx context.Context,
	env proto.MissingTransactionsEnvelope,
) (proto.RepairedTransactions, error) {

	txs, found := x.submitted.LookupTxs(ctx, env.Missing_transactions.Transaction_hashes)
	if !found {
		return proto.RepairedTransactions{}, nil
	}

	return proto.RepairedTransactions{
		Transactions: protoutil.Pack(txs),
	}, nil
}
