package repairservice

import (
	"context"
	"fmt"
	"sync"

	"github.com/groundhogchain/chuckd/sub/catchup"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/sub/scp/scpbroker"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairService struct {
	conn        sched.SchedulerCoordConn
	scpBroker   *scpbroker.SCPBroker //XXX: use to repair missing scp envelopes
	cvpool      cvpool.CVPool        // use to repair missing consensus values
	cbpool      catchup.CBPool       // use to repair missing transactions
	schedReady  chan struct{}
	brokerReady chan struct{}
	//
	lk        sync.Mutex
	nominated map[proto.ConsensusValueHash]proto.BlockHeight
}

func NewRepairService(
	cvpool cvpool.CVPool,
	cbpool catchup.CBPool,
) *RepairService {
	return &RepairService{
		cvpool:      cvpool,
		cbpool:      cbpool,
		schedReady:  make(chan struct{}),
		brokerReady: make(chan struct{}),
		nominated:   map[proto.ConsensusValueHash]proto.BlockHeight{},
	}
}

func (x *RepairService) Stop() {}

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

func (x *RepairService) AnnounceNomination(height proto.BlockHeight, cvh proto.ConsensusValueHash) {
	x.lk.Lock()
	defer x.lk.Unlock()

	// clear older nominations
	for mHash, mHeight := range x.nominated {
		if mHeight < height {
			delete(x.nominated, mHash)
		}
	}

	x.nominated[cvh] = height
}

func (x *RepairService) isOurNomination(cvh proto.ConsensusValueHash) bool {
	x.lk.Lock()
	defer x.lk.Unlock()
	_, ok := x.nominated[cvh]
	return ok
}

func (x *RepairService) Repair_missing_txs(
	ctx context.Context,
	env proto.MissingTransactionsEnvelope,
) (rtx proto.RepairedTransactions, err error) {

	ctx = liblog.Enterf(ctx, "Repair_missing_txs", "by=%s; height=%d; cv_hash=%s; count=%d",
		libxdr.HashString(env.Signer), env.Missing_transactions.Cue.Block_height,
		libxdr.HashString(env.Missing_transactions.Cue.Cv_hash),
		len(env.Missing_transactions.Transaction_hashes))
	defer func() {
		liblog.Leavef(ctx, "by=%s; height=%d; cv_hash=%s; count=%d; error=%v; found=%d",
			libxdr.HashString(env.Signer), env.Missing_transactions.Cue.Block_height,
			libxdr.HashString(env.Missing_transactions.Cue.Cv_hash),
			len(env.Missing_transactions.Transaction_hashes),
			err, len(rtx.Transactions))
	}()

	<-x.schedReady // wait for x.conn != nil

	txs, err := x.repairMissingTxsFromMempool(ctx, env.Missing_transactions.Transaction_hashes)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_txs not found in mempool (%v): height=%d; cv_hash=%s",
			err, env.Missing_transactions.Cue.Block_height,
			libxdr.HashString(env.Missing_transactions.Cue.Cv_hash))
	} else {
		return txs, nil
	}

	txs, err = x.repairMissingTxsFromStorage(ctx, env)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_txs not found in storage (%v): height=%d; cv_hash=%s",
			err, env.Missing_transactions.Cue.Block_height,
			libxdr.HashString(env.Missing_transactions.Cue.Cv_hash))
		return proto.RepairedTransactions{}, err
	}

	return txs, nil
}

func (x *RepairService) repairMissingTxsFromStorage(
	ctx context.Context,
	env proto.MissingTransactionsEnvelope,
) (rtx proto.RepairedTransactions, err error) {

	ctx = liblog.Enterf(ctx, "repairMissingTxsFromStorage", "height=%d; cv_hash=%s; count=%d",
		env.Missing_transactions.Cue.Block_height,
		libxdr.HashString(env.Missing_transactions.Cue.Cv_hash),
		len(env.Missing_transactions.Transaction_hashes))
	defer func() {
		liblog.Leavef(ctx, "height=%d; cv_hash=%s; count=%d; error=%v; found=%d",
			env.Missing_transactions.Cue.Block_height,
			libxdr.HashString(env.Missing_transactions.Cue.Cv_hash),
			len(env.Missing_transactions.Transaction_hashes),
			err, len(rtx.Transactions))
	}()

	cb := x.cbpool.CBPoolLookup(ctx, env.Missing_transactions.Cue)
	if cb == nil {
		liblog.Errorf(ctx, "catchup block not found in storage: cv_hash=%s",
			libxdr.HashString(env.Missing_transactions.Cue.Cv_hash))
		return proto.RepairedTransactions{}, fmt.Errorf("catchup block not found")
	}

	txs, err := xdrutil.SerializeSignedTransactionBatch(cb.Signed_transactions)
	if err != nil {
		liblog.Errorf(ctx, "failed to serialize transaction batch: cv_hash=%s",
			libxdr.HashString(env.Missing_transactions.Cue.Cv_hash))
		return proto.RepairedTransactions{},
			fmt.Errorf("failed to serialize transaction batch (%w)", err)
	}

	txhs, err := xdrutil.SerializeSignedTransactionHashBatch(cb.Consensus_value.Nomination.Transactions_hash)
	if err != nil {
		liblog.Errorf(ctx, "failed to serialize transaction hash batch: cv_hash=%s",
			libxdr.HashString(env.Missing_transactions.Cue.Cv_hash))
		return proto.RepairedTransactions{},
			fmt.Errorf("failed to serialize transaction hash batch (%w)", err)
	}

	if len(txs) != len(txhs) {
		liblog.Errorf(ctx, "transaction and hash count mismatch: cv_hash=%s",
			libxdr.HashString(env.Missing_transactions.Cue.Cv_hash))
		return proto.RepairedTransactions{},
			fmt.Errorf("transaction and hash count mismatch: %d != %d",
				len(txs), len(txhs))
	}

	m := map[proto.SignedTransactionHash]int{}
	for i, txh := range txhs {
		m[txh] = i
	}

	result := make([]proto.SignedTransaction, len(env.Missing_transactions.Transaction_hashes))
	for i, q := range env.Missing_transactions.Transaction_hashes {
		j, ok := m[q]
		if !ok {
			liblog.Errorf(ctx, "transaction hash not found: %s", libxdr.HashString(q))
			return proto.RepairedTransactions{},
				fmt.Errorf("transaction hash not found: %s", libxdr.HashString(q))
		}
		result[i] = txs[j]
	}

	return proto.RepairedTransactions{Transactions: result}, nil
}

func (x *RepairService) repairMissingTxsFromMempool(
	ctx context.Context,
	txhs []proto.SignedTransactionHash,
) (proto.RepairedTransactions, error) {

	found, err := x.conn.Schedcoord_lookup_transactions(ctx, txhs)
	if err != nil {
		return proto.RepairedTransactions{}, err
	}
	must.Assert(ctx, len(found) == len(txhs))

	txs := make([]proto.SignedTransaction, len(txhs))
	for i, otx := range found {
		// we must have all transactions that we have nominated, or else
		// the request is invalid
		if otx == nil {
			return proto.RepairedTransactions{}, fmt.Errorf("not all transactions found in mempool")
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
	cvc proto.ConsensusValueCue,
) (proto.OptConsensusValue, error) {

	liblog.Debugf(ctx, "repair_missing_cv request: height=%d; cv_hash=%s", cvc.Block_height, libxdr.HashString(cvc.Cv_hash))

	// straggler validators may need a consensus value after
	// it has been externalized and committed at other nodes.
	// in this case, the consensus value may no longer be in the
	// cvpool.
	//
	// thus, when a value is not found in the cvpool, repair
	// tries to find it in the storage layer, which persist
	// all consensus values seen by the node during the slot round.
	//
	// this helps stragglers who:
	// - replay old scp envelopes, henceforth need to validate old nominations,
	// henceforth need to repair old consensus values, or
	// - process externalizations whose rounds they didn't participate in,
	// henceforth they need to repair the missing consensus values.
	//
	// requests for missing cvs will block the repair attempts forever,
	// and they need to be aborted at the caller when they discover newer externalization have happened.

	cvx, err := x.cvpool.CVPoolLookup(ctx, cvc)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_cv failed to lookup cv in pool: %v", err)
	}
	if cvx != nil {
		liblog.Infof(ctx, "repair_missing_cv serving from pool: height=%d; cv_hash=%s",
			cvc.Block_height, libxdr.HashString(cvc.Cv_hash))
		return cvx, nil
	}

	cv, err := x.cvpool.CVPoolFind(ctx, cvc)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_cv failed to find cv in pool: %v", err)
		return nil, err
	}
	if cv != nil {
		liblog.Infof(ctx, "repair_missing_cv serving from storage: height=%d; cv_hash=%s",
			cvc.Block_height, libxdr.HashString(cvc.Cv_hash))
	} else {
		liblog.Errorf(ctx, "repair_missing_cv not found: height=%d; cv_hash=%s",
			cvc.Block_height, libxdr.HashString(cvc.Cv_hash))
	}
	return cv, nil
}

// XXX: dos protection
func (x *RepairService) Repair_missing_scp_envelopes(
	ctx context.Context,
	slot proto.SCPSlotIndex,
) (proto.SCPEnvelopes, error) {
	<-x.brokerReady

	liblog.Debugf(ctx, "repair missing scp envelopes request, for slot %d", slot)
	envs, err := x.scpBroker.RepairEnvelopes(ctx, slot)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_scp_envelopes failed to repair envelopes: %v", err)
		return nil, err
	}
	return envs, nil
}
