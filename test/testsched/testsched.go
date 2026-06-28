package testsched

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/test/testdb"
	"github.com/groundhogchain/chuckd/test/testmempool"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/librand"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
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

func (x *TestScheduler) Schedulercoord_nominate(
	ctx context.Context,
	onto proto.DbTip,
) (proto.SchedulerNominationResult, error) {

	blockID := x.generateBlockID()

	<-x.ready // wait for x.coord != nil

	x.db.CreateSnapshot(ctx, onto.Id, blockID, proto.KBlockExecutionModeProposal)

	txs := x.mempool.DrainTxs()

	txnds := make([]proto.NondeterministicProposalData, len(txs))
	for i, tx := range txs {
		x.db.AddSignedTx(ctx, blockID, tx)
		txnds[i] = testutil.TestNondeterministicData
	}

	BlockHash := x.db.CloseSnapshot(ctx, blockID)

	nomination := proto.Nomination{
		Block_height:    onto.Height + 1,
		Block_hash:      BlockHash,
		Prev_block_hash: onto.Block_hash,
		Transactions_hash: xdrutil.ParallelizeSignedTransactionHashBatch(
			x.m,
			systemfixed.NominationSignedTransactionHashBatchMaxSegments,
			protoutil.HashSignedTxs(protoutil.Pack(txs))),
		Transactions_nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(
			x.m,
			systemfixed.NominationNondeterministicDataBatchMaxSegments,
			txnds),
	}

	r := proto.SchedulerNominationResult{}
	r.Outcome = proto.SchedulerNominationOK
	*r.Nominated_block() = proto.SchedulerNominatedBlock{
		Block_id:   blockID,
		Nomination: nomination,
		Txs: xdrutil.ParallelizeSignedTransactionBatch(x.m,
			systemfixed.CatchupTxBatchMaxSegments, protoutil.Pack(txs)),
	}

	return r, nil
}

func (x *TestScheduler) Schedulercoord_validate(ctx context.Context, onto proto.DbTip, nomination proto.Nomination) (proto.SchedulerValidationResult, error) {

	nomTxhs, err := xdrutil.SerializeSignedTransactionHashBatch(nomination.Transactions_hash)
	if err != nil {
		return proto.SchedulerValidationResult{}, err
	}
	nomNDs, err := xdrutil.SerializeNondeterministicProposalDataBatch(nomination.Transactions_nondeterministic_data)
	if err != nil {
		return proto.SchedulerValidationResult{}, err
	}

	blockID := x.generateBlockID()
	x.db.CreateSnapshot(ctx, onto.Id, blockID, proto.KBlockExecutionModeValidation)

	txs, missingTxHashes, _ := x.mempool.LookupTxs(nomTxhs)
	if len(missingTxHashes) > 0 {
		x.db.AbandonSnapshot(ctx, blockID)
		r := proto.SchedulerValidationResult{
			Outcome: proto.SchedulerValidationMissingTxs,
		}
		*r.Missing_tx_hashes() = missingTxHashes
		return r, nil
	}

	for i, tx := range txs {
		ptx := &proto.ProposedTransaction{
			Signed_tx:                      tx,
			Nondeterministic_proposal_data: nomNDs[i],
		}
		x.db.AddProposedTx(ctx, blockID, ptx)
	}

	blockHash := x.db.CloseSnapshot(ctx, blockID)

	r := proto.SchedulerValidationResult{
		Outcome: proto.SchedulerValidationOK,
	}
	*r.Block() = proto.SchedulerValidationBlock{
		Db_tip: proto.DbTip{
			Height:     onto.Height + 1,
			Block_hash: blockHash,
			Id:         blockID,
		},
		Signed_transactions: xdrutil.ParallelizeSignedTransactionBatch(
			x.m,
			systemfixed.CatchupTxBatchMaxSegments,
			txs),
	}
	return r, nil
}

func (x *TestScheduler) Schedulercoord_commit_block(ctx context.Context, blockID proto.BlockID) (proto.SchedulerCommitOutcome, error) {
	x.db.OpenSnapshot(ctx, blockID)
	_, hashes := x.db.CommitSnapshot(ctx, blockID)
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

func (x *TestScheduler) Schedcoord_pending_transaction_count(ctx context.Context) (int64, error) {
	return int64(x.mempool.Len()), nil
}
