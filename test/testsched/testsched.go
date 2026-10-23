package testsched

import (
	"context"
	"math/rand"
	"sync"

	"github.com/thruputchain/chuckd/sub/sched"
	"github.com/thruputchain/chuckd/system/systemfixed"
	"github.com/thruputchain/chuckd/test/testdb"
	"github.com/thruputchain/chuckd/test/testmempool"
	"github.com/thruputchain/chuckd/test/testutil"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/chuckd/util/xdrutil"
	"github.com/thruputchain/lib-go/librand"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type TestScheduler struct {
	m       libxdr.XdrMarshaler
	signer  proto.ValidatorPublicInfo
	mempool *testmempool.TestMempool
	db      *testdb.TestDb
	//
	coord sched.CoordSchedulerConn
	ready chan struct{}
	//
	lk     sync.Mutex
	introd bool
	rand   *rand.Rand
}

func NewTestScheduler(
	ctx context.Context,
	signer proto.ValidatorPublicInfo,
	genesisTip proto.DbTip,
	dbDir string,
) *TestScheduler {
	return &TestScheduler{
		m:       libxdr.NewXdrMarshaler(),
		signer:  signer,
		db:      testdb.NewTestDb(ctx, genesisTip, dbDir),
		ready:   make(chan struct{}),
		mempool: testmempool.NewTestMempool(),
		rand:    librand.NewRand(),
	}
}

func (x *TestScheduler) SetCoordScheduler(conn sched.CoordSchedulerConn) {
	x.coord = conn
	close(x.ready)
}

func (x *TestScheduler) generateBlockID() proto.BlockID {
	x.lk.Lock()
	defer x.lk.Unlock()
	return proto.BlockID(x.rand.Int63())
}

func (x *TestScheduler) Schedulercoord_null(_ context.Context) error {
	return nil
}

func (x *TestScheduler) Schedulercoord_intro_coord(ctx context.Context, card proto.CoordCard) (proto.CoordinatorRegistrationOutcome, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	must.Assert(ctx, !x.introd)
	x.introd = true
	return proto.CoordinatorRegistrationOK, nil
}

func (x *TestScheduler) AddTxs(ctx context.Context, txs ...*proto.SignedTransaction) {
	x.mempool.AddTxs(protoutil.Pack(txs))
}

func (x *TestScheduler) Schedulercoord_submit_unvetted(ctx context.Context, batch proto.ParallelSignedTransactionBatch) error {
	go func() {
		<-x.ready // wait for x.coord != nil

		err := x.coord.Coordscheduler_notify_vetted(ctx, batch)
		must.NoError(ctx, err)
	}()

	txs, err := xdrutil.SerializeSignedTransactionBatch(batch)
	must.NoError(ctx, err)
	x.mempool.AddTxs(txs)
	return nil
}

func (x *TestScheduler) Schedulercoord_submit_vetted(ctx context.Context, vid proto.ValidatorID, batch proto.ParallelSignedTransactionBatch) error {
	txs, err := xdrutil.SerializeSignedTransactionBatch(batch)
	must.NoError(ctx, err)
	x.mempool.AddTxs(txs)
	return nil
}

func (x *TestScheduler) Schedcoord_lookup_transactions(
	ctx context.Context,
	hashes proto.SignedTransactionHashBatch,
) (proto.OptSignedTransactionBatch, error) {
	_, _, mixed := x.mempool.LookupTxs(hashes)
	return mixed, nil
}

func (x *TestScheduler) Schedcoord_lookup_transaction_outcomes(
	ctx context.Context,
	hashes proto.TransactionHashBatch,
) (proto.SchedulerTransactionOutcomeBatch, error) {
	_, _, mixed := x.mempool.LookupTxs(hashes)
	outcomes := make(proto.SchedulerTransactionOutcomeBatch, len(hashes))
	for i, tx := range mixed {
		if tx != nil {
			outcomes[i].Status = proto.SchedulerTransactionPending
		} else {
			outcomes[i].Status = proto.SchedulerTransactionNotFound
		}
	}
	return outcomes, nil
}

func (x *TestScheduler) Schedulercoord_propose_block(
	ctx context.Context,
	onto proto.BlockID,
	propCx proto.ProposalContext,
) (proto.SchedulerProposalResult, error) {
	must.Assert(ctx, propCx.Protocol == proto.Protocol(0))

	blockID := x.generateBlockID()

	<-x.ready // wait for x.coord != nil

	height := x.db.CreateSnapshot(ctx, onto, blockID, proto.KBlockExecutionModeProposal)

	txs := x.mempool.DrainTxs()

	txnds := make([]proto.NondeterministicProposalData, len(txs))
	for i, tx := range txs {
		x.db.AddSignedTx(ctx, blockID, tx)
		txnds[i] = testutil.TestNondeterministicData
	}

	storageHash := x.db.CloseSnapshot(ctx, blockID)

	txHashes := protoutil.HashSignedTxs(protoutil.Pack(txs))

	proposal := proto.Proposal{
		Header: proto.ProposalHeader{
			Network_hash:    testutil.TestNetworkHash,
			Proposer:        propCx.Proposer,
			Block_timepoint: propCx.Block_timepoint,
			Block_height:    height,
			Txs_hash:        testutil.TxsHashOf(txHashes),
			Storage_hash:    storageHash,
			Events_hash:     testutil.TestEventsHash,
		},
		Transactions: proto.ProposalTxs{
			Signed_hash: xdrutil.ParallelizeSignedTransactionHashBatch(
				x.m,
				systemfixed.NominationTransactionHashBatchMaxSegments,
				txHashes),
			Nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(
				x.m,
				systemfixed.NominationNondeterministicDataBatchMaxSegments,
				txnds),
		},
	}

	r := proto.SchedulerProposalResult{}
	r.Outcome = proto.SchedulerProposalOK
	*r.Proposed_block() = proto.SchedulerProposedBlock{
		Block_id: blockID,
		Proposal: proposal,
		Txs: xdrutil.ParallelizeSignedTransactionBatch(x.m,
			systemfixed.CatchupTxBatchMaxSegments, protoutil.Pack(txs)),
	}

	return r, nil
}

func (x *TestScheduler) Schedulercoord_validate_block(ctx context.Context, onto proto.BlockID, proposal proto.Proposal) (proto.SchedulerValidationResult, error) {

	nomTxhs, err := xdrutil.SerializeSignedTransactionHashBatch(proposal.Transactions.Signed_hash)
	if err != nil {
		return proto.SchedulerValidationResult{}, err
	}
	nomNDs, err := xdrutil.SerializeNondeterministicProposalDataBatch(proposal.Transactions.Nondeterministic_data)
	if err != nil {
		return proto.SchedulerValidationResult{}, err
	}

	blockID := x.generateBlockID()
	height := x.db.CreateSnapshot(ctx, onto, blockID, proto.KBlockExecutionModeValidation)

	txs, missingTxHashes, _ := x.mempool.LookupTxs(nomTxhs)
	if len(missingTxHashes) > 0 {
		x.db.AbandonSnapshot(ctx, blockID)
		r := proto.SchedulerValidationResult{
			Outcome: proto.SchedulerValidationMissingTxs,
		}
		*r.Missing_signed_tx_hashes() = missingTxHashes
		return r, nil
	}

	for i, tx := range txs {
		ptx := &proto.ProposedTransaction{
			Signed_tx:                      tx,
			Nondeterministic_proposal_data: nomNDs[i],
		}
		x.db.AddProposedTx(ctx, blockID, ptx)
	}

	storageHash := x.db.CloseSnapshot(ctx, blockID)

	// recompute the post-execution header and check it agrees with the proposal
	computed := proto.ProposalHeader{
		Network_hash:    proposal.Header.Network_hash,
		Proposer:        proposal.Header.Proposer,
		Block_timepoint: proposal.Header.Block_timepoint,
		Block_height:    height,
		Txs_hash:        testutil.TxsHashOf(nomTxhs),
		Storage_hash:    storageHash,
		Events_hash:     testutil.TestEventsHash,
	}
	if !protoutil.HeaderEquals(&computed, &proposal.Header) {
		return proto.SchedulerValidationResult{
			Outcome: proto.SchedulerValidationInvalid,
		}, nil
	}

	r := proto.SchedulerValidationResult{
		Outcome: proto.SchedulerValidationOK,
	}
	*r.Block() = proto.SchedulerValidationBlock{
		Block_id: blockID,
		Signed_transactions: xdrutil.ParallelizeSignedTransactionBatch(
			x.m,
			systemfixed.CatchupTxBatchMaxSegments,
			txs),
	}
	return r, nil
}

func (x *TestScheduler) Schedulercoord_commit_block(ctx context.Context, blockID proto.BlockID, value proto.ConsensusValue) (proto.SchedulerCommitOutcome, error) {
	x.db.OpenSnapshot(ctx, blockID)
	_, hashes := x.db.CommitSnapshot(ctx, blockID, value)
	x.mempool.RemoveTxs(hashes)
	return proto.SchedulerCommitOK, nil
}

func (x *TestScheduler) Schedcoord_get_tip(ctx context.Context) (proto.SchedulerGetTipResult, error) {
	r := proto.SchedulerGetTipResult{}
	r.Outcome = proto.SchedulerGetTipOK
	*r.Db_tip() = x.db.GetTip()
	return r, nil
}

func (x *TestScheduler) Schedcoord_load_genesis_configuration(context.Context, proto.GenesisConfiguration) (proto.DbSchedResult, error) {
	panic("not used")
}

func (x *TestScheduler) Schedulercoord_commit_genesis_ceremony(context.Context, proto.GenesisCeremonyRequest) (proto.GenesisCeremonyResult, error) {
	panic("not used")
}

func (x *TestScheduler) Schedcoord_pending_transaction_count(ctx context.Context) (int64, error) {
	return int64(x.mempool.Len()), nil
}
