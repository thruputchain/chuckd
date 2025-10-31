package repairservice

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/sub/scp/scpbroker"
	"github.com/groundhogchain/chuckd/sub/storage"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairService struct {
	conn        sched.SchedulerCoordConn
	broker      *scpbroker.SCPBroker       //XXX: use to repair missing scp envelopes
	cvpool      *cvpool.ConsensusValuePool // use to repair missing consensus values
	storage     *storage.DiskStorage
	schedReady  chan struct{}
	brokerReady chan struct{}
}

func NewRepairService(
	cvpool *cvpool.ConsensusValuePool,
	storage *storage.DiskStorage,
) *RepairService {
	return &RepairService{
		cvpool:      cvpool,
		storage:     storage,
		schedReady:  make(chan struct{}),
		brokerReady: make(chan struct{}),
	}
}

func (x *RepairService) SetSchedConn(conn sched.SchedulerCoordConn) {
	x.conn = conn
	close(x.schedReady)
}

func (x *RepairService) SetSCPBroker(broker *scpbroker.SCPBroker) {
	x.broker = broker
	close(x.brokerReady)
}

func (x *RepairService) Repair_null(ctx context.Context) error {
	return nil
}

// XXX: verify block_hash is our nomination; verify requester is worth responding to, and their signature
func (x *RepairService) Repair_missing_txs(
	ctx context.Context,
	env proto.MissingTransactionsEnvelope,
) (proto.RepairedTransactions, error) {
	<-x.schedReady // wait for x.conn != nil

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

// XXX: needs dos protection
func (x *RepairService) Repair_missing_cv(
	ctx context.Context,
	cvh proto.ConsensusValueHash,
) (proto.OptConsensusValue, error) {

	return x.cvpool.ConsensusValuePoolLookup(ctx, cvh), nil
}

// XXX: dos protection
func (x *RepairService) Repair_missing_scp_envelopes(
	ctx context.Context,
	since proto.SCPSlotIndex,
) (proto.SCPEnvelopes, error) {

	return x.storage.FindSCPEnvelopesSinceSlot(ctx, since), nil
}
