package repairservice

import (
	"context"
	"fmt"
	"sync"

	"github.com/groundhogchain/chuckd/sub/catchup"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/sub/scp/scpbroker"
	"github.com/groundhogchain/chuckd/util/protoutil"
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
	nominated map[protoutil.ValueHash]proto.BlockHeight
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
		nominated:   map[protoutil.ValueHash]proto.BlockHeight{},
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

func (x *RepairService) AnnounceNomination(value *proto.ConsensusValue) {
	x.lk.Lock()
	defer x.lk.Unlock()

	// clear older nominations
	for mHash, mHeight := range x.nominated {
		if mHeight < value.Proposal_header.Block_height {
			delete(x.nominated, mHash)
		}
	}

	x.nominated[protoutil.HashValue(value)] = value.Proposal_header.Block_height
}

func (x *RepairService) Repair_missing_txs(
	ctx context.Context,
	env proto.MissingTransactionsEnvelope,
) (rtx proto.RepairedTransactions, err error) {

	ctx = liblog.Enterf(ctx, "Repair_missing_txs", "by=%s; height=%d; value=%s; count=%d",
		libxdr.HashString(env.Signer), env.Missing_transactions.Consensus_value.Proposal_header.Block_height,
		protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value),
		len(env.Missing_transactions.Transaction_hashes))
	defer func() {
		liblog.Leavef(ctx, "by=%s; height=%d; value=%s; count=%d; error=%v; found=%d",
			libxdr.HashString(env.Signer), env.Missing_transactions.Consensus_value.Proposal_header.Block_height,
			protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value),
			len(env.Missing_transactions.Transaction_hashes),
			err, len(rtx.Transactions))
	}()

	<-x.schedReady // wait for x.conn != nil

	txs, err := x.repairMissingTxsFromMempool(ctx, env.Missing_transactions.Transaction_hashes)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_txs not found in mempool (%v): height=%d; value=%s",
			err, env.Missing_transactions.Consensus_value.Proposal_header.Block_height,
			protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value))
	} else {
		return txs, nil
	}

	txs, err = x.repairMissingTxsFromStorage(ctx, env)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_txs not found in storage (%v): height=%d; value=%s",
			err, env.Missing_transactions.Consensus_value.Proposal_header.Block_height,
			protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value))
		return proto.RepairedTransactions{}, err
	}

	return txs, nil
}

func (x *RepairService) repairMissingTxsFromStorage(
	ctx context.Context,
	env proto.MissingTransactionsEnvelope,
) (rtx proto.RepairedTransactions, err error) {

	ctx = liblog.Enterf(ctx, "repairMissingTxsFromStorage", "height=%d; value=%s; count=%d",
		env.Missing_transactions.Consensus_value.Proposal_header.Block_height,
		protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value),
		len(env.Missing_transactions.Transaction_hashes))
	defer func() {
		liblog.Leavef(ctx, "height=%d; value=%s; count=%d; error=%v; found=%d",
			env.Missing_transactions.Consensus_value.Proposal_header.Block_height,
			protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value),
			len(env.Missing_transactions.Transaction_hashes),
			err, len(rtx.Transactions))
	}()

	cb, err := x.cbpool.CBPoolLookup(ctx, env.Missing_transactions.Consensus_value)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_txs failed to lookup catchup block: %v", err)
		return proto.RepairedTransactions{}, err
	}
	if cb == nil {
		liblog.Errorf(ctx, "catchup block not found in storage: value=%s",
			protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value))
		return proto.RepairedTransactions{}, fmt.Errorf("catchup block not found")
	}

	txs, err := xdrutil.SerializeSignedTransactionBatch(cb.Signed_transactions)
	if err != nil {
		liblog.Errorf(ctx, "failed to serialize transaction batch: value=%s",
			protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value))
		return proto.RepairedTransactions{},
			fmt.Errorf("failed to serialize transaction batch (%w)", err)
	}

	txhs, err := xdrutil.SerializeSignedTransactionHashBatch(cb.Consensus_basis.Proposal_txs.Hash)
	if err != nil {
		liblog.Errorf(ctx, "failed to serialize transaction hash batch: value=%s",
			protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value))
		return proto.RepairedTransactions{},
			fmt.Errorf("failed to serialize transaction hash batch (%w)", err)
	}

	if len(txs) != len(txhs) {
		liblog.Errorf(ctx, "transaction and hash count mismatch: value=%s",
			protoutil.ValueDisplay(&env.Missing_transactions.Consensus_value))
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
func (x *RepairService) Repair_missing_proposal(
	ctx context.Context,
	value proto.ConsensusValue,
) (proto.OptProposal, error) {

	liblog.Debugf(ctx, "repair_missing_proposal request: height=%d; value=%s",
		value.Proposal_header.Block_height, protoutil.ValueDisplay(&value))

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

	basisX, err := x.cvpool.CVPoolLookup(ctx, value)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_proposal failed to lookup cv in pool: %v", err)
	}
	if basisX != nil {
		liblog.Infof(ctx, "repair_missing_proposal serving from pool: height=%d; value=%s",
			value.Proposal_header.Block_height, protoutil.ValueDisplay(&value))
		return proposalFromBasis(basisX), nil
	}

	basis, err := x.cvpool.CVPoolFind(ctx, value)
	if err != nil {
		liblog.Errorf(ctx, "repair_missing_proposal failed to find cv in pool: %v", err)
		return nil, err
	}
	if basis != nil {
		liblog.Infof(ctx, "repair_missing_proposal serving from storage: height=%d; value=%s",
			value.Proposal_header.Block_height, protoutil.ValueDisplay(&value))
		return proposalFromBasis(basis), nil
	}
	liblog.Errorf(ctx, "repair_missing_proposal not found: height=%d; value=%s",
		value.Proposal_header.Block_height, protoutil.ValueDisplay(&value))
	return nil, nil
}

func proposalFromBasis(basis *proto.ConsensusBasis) *proto.Proposal {
	return &proto.Proposal{
		Header:       basis.Value.Proposal_header,
		Transactions: basis.Proposal_txs,
	}
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
