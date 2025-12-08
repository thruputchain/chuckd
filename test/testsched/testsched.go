package testsched

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/test/testdb"
	"github.com/groundhogchain/chuckd/test/testmempool"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
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

func NewTestScheduler(ctx context.Context, signer proto.ValidatorPublicInfo, tip proto.DbTip) *TestScheduler {
	return &TestScheduler{
		m:       libxdr.NewXdrMarshaler(),
		signer:  signer,
		db:      testdb.NewTestDb(ctx, tip),
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

func (x *TestScheduler) Schedulercoord_submit_unvetted(ctx context.Context, txs proto.SignedTransactionBatch) error {
	go func() {
		<-x.ready // wait for x.coord != nil

		err := x.coord.Coordscheduler_notify_vetted(ctx, txs)
		must.NoError(ctx, err)
	}()

	x.mempool.AddTxs(txs)
	return nil
}

func (x *TestScheduler) Schedulercoord_submit_vetted(ctx context.Context, vid proto.ValidatorID, txs proto.SignedTransactionBatch) error {
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

	go func() {
		<-x.ready // wait for x.coord != nil

		x.db.CreateSnapshot(ctx, onto.Id, blockID, proto.KBlockExecutionModeProposal)

		txs := x.mempool.DrainTxs()

		txnoms := make([]proto.NominatedTransaction, len(txs))
		for i, tx := range txs {
			x.db.AddSignedTx(ctx, blockID, tx)
			txnoms[i] = proto.NominatedTransaction{
				Nondeterministic_proposal_data: testutil.TestNondeterministicData,
			}
		}

		blockHash := x.db.CloseSnapshot(ctx, blockID)

		nomination := proto.Nomination{
			Proposer:                x.signer,
			Block_height:            onto.Height + 1,
			Block_hash:              blockHash,
			Prev_block_hash:         onto.Hash,
			Transaction_hashes:      protoutil.HashSignedTxs(x.m, protoutil.Pack(txs)),
			Transaction_nominations: txnoms,
		}

		chunk := proto.SchedulerNominationChunk{
			Chunk_idx:    0,
			Total_chunks: 1,
			Chunk: proto.SchedulerNomination{
				Block_id:   blockID,
				Nomination: nomination,
			},
		}

		err := x.coord.Coordscheduler_finish_nomination(ctx, chunk)
		must.NoError(ctx, err)
	}()

	r := proto.SchedulerNominationResult{}
	r.Outcome = proto.SchedulerNominationOK
	*r.Block_id() = blockID

	return r, nil
}

func (x *TestScheduler) Schedulercoord_validate(ctx context.Context, onto proto.DbTip, nomination proto.Nomination) (proto.SchedulerValidationResult, error) {

	blockID := x.generateBlockID()
	x.db.CreateSnapshot(ctx, onto.Id, blockID, proto.KBlockExecutionModeValidation)

	txs, missingTxHashes, _ := x.mempool.LookupTxs(nomination.Transaction_hashes)
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
			Nondeterministic_proposal_data: nomination.Transaction_nominations[i].Nondeterministic_proposal_data,
		}
		x.db.AddProposedTx(ctx, blockID, ptx)
	}

	blockHash := x.db.CloseSnapshot(ctx, blockID)

	r := proto.SchedulerValidationResult{
		Outcome: proto.SchedulerValidationOK,
	}
	*r.Db_tip() = proto.DbTip{
		Height: onto.Height + 1,
		Hash:   blockHash,
		Id:     blockID,
	}
	return r, nil
}

func (x *TestScheduler) Schedulercoord_commit_block(ctx context.Context, blockID proto.BlockID) (proto.SchedulerCommitOutcome, error) {
	x.db.OpenSnapshot(ctx, blockID)
	_, hashes := x.db.CommitSnapshot(ctx, blockID)
	x.mempool.RemoveTxs(hashes)
	return proto.SchedulerCommitOK, nil
}

func (x *TestScheduler) Schedcoord_get_tip(ctx context.Context) (proto.DbTip, error) {
	return x.db.GetTip(), nil
}

func (x *TestScheduler) Schedcoord_load_genesis_block(context.Context, proto.GenesisBlock) (proto.DbSchedResult, error) {
	panic("not used")
}
