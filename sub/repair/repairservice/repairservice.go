package repairservice

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/sub/scp/scpbroker"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairService struct {
	conn        sched.SchedulerCoordConn
	scpBroker   *scpbroker.SCPBroker       //XXX: use to repair missing scp envelopes
	cvpool      *cvpool.ConsensusValuePool // use to repair missing consensus values
	schedReady  chan struct{}
	brokerReady chan struct{}
}

func NewRepairService(
	cvpool *cvpool.ConsensusValuePool,
) *RepairService {
	return &RepairService{
		cvpool:      cvpool,
		schedReady:  make(chan struct{}),
		brokerReady: make(chan struct{}),
	}
}

func (x *RepairService) SetSchedConn(conn sched.SchedulerCoordConn) {
	x.conn = conn
	close(x.schedReady)
}

func (x *RepairService) SetSCPBroker(broker *scpbroker.SCPBroker) {
	x.scpBroker = broker
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
	liblog.Debugf("repair missing transactions request, count %d", len(env.Missing_transactions.Transaction_hashes))

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
	height proto.BlockHeight,
) (proto.OptConsensusValue, error) {

	liblog.Debugf("repair missing consensus value request, cv hash %s", libxdr.HashString(cvh))

	// straggler validators may need a consensus value after
	// it has been externalized and committed at other nodes.
	// in this case, the consensus value may no longer be in the
	// cvpool.
	//
	// thus, when a value is not found in the cvpool, repair
	// tries to find it in the storage layer, which only persist
	// externalized consensus values.
	//
	// requests for non-externalized past cvs will block the repair attempts forever,
	// and they need to be aborted at the caller when they discover newer externalization have happened.
	// requests for externalized cvs will be answered by the storage layer, which will allow
	// externalization messages to be processed at the called, and thus externalizations to be acted on.

	cvx := x.cvpool.ConsensusValuePoolLookupHash(ctx, cvh)
	if cvx != nil {
		liblog.Infof("serving missing cv from pool, hash=%s height=%d", libxdr.HashString(cvh), height)
		return cvx, nil
	}

	cv := x.cvpool.ConsensusValuePoolFindHeight(ctx, height)
	if cv != nil {
		if protoutil.HashCVFast(cv) != cvh {
			cv = nil // the requested cv was not externalized
			liblog.Infof("requested missing cv was not externalized, hash=%s height=%d", libxdr.HashString(cvh), height)
		} else {
			liblog.Infof("serving missing cv from storage, hash=%s height=%d", libxdr.HashString(cvh), height)
		}
	}
	return cv, nil
}

// XXX: dos protection
func (x *RepairService) Repair_missing_scp_envelopes(
	ctx context.Context,
	slot proto.SCPSlotIndex,
) (proto.SCPEnvelopes, error) {
	<-x.brokerReady

	liblog.Debugf("repair missing scp envelopes request, for slot %d", slot)
	return x.scpBroker.RepairEnvelopes(ctx, slot), nil
}
