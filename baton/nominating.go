package baton

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func canNominate(nominate *Tip, nominating *nominating) bool {
	if nominate == nil {
		return false
	}
	if nominating == nil {
		return true
	}
	return nominate.Height > nominating.Height
}

func (b *Baton) nominate(ctx context.Context, tip Tip) (*proto.Proposal, proto.BlockID, error) {

	// pool all available transactions (our own + from batches) excluding already committed ones
	uncommittedTxs := []*proto.SignedTransaction{}
	for _, stx := range b.handler.ListTxs(ctx) {
		hash := sha256.Sum256(xdrutil.MarshalXdr(stx))
		if !b.handler.IsCommittedTx(ctx, hash) {
			uncommittedTxs = append(uncommittedTxs, stx)
		}
	}

	// prioritize and pick the top, applying any limits on transactions per block
	txsToNominate := b.handler.PrioritizeTxs(ctx, uncommittedTxs)
	if len(txsToNominate) == 0 {
		return nil, 0, fmt.Errorf("nothing to nominate")
	}

	// start working on proposing a block, while being ready to cancel if a block is externalized by someone else
	nom, err := b.prepareNomination(ctx, tip, txsToNominate)
	if err != nil {
		// XXX: ctx may be cancelled
		// XXX: ErrNoExecs
		return nil, 0, err
	}

	// construct proposed block
	batch, batchHash := protoutil.PrepareBatch(nom.proposedTxs)
	proposal := &proto.Proposal{
		Block_height:                 nom.blockHeight,
		Block_hash:                   nom.blockHash,
		Prev_block_hash:              nom.prevBlockHash,
		Proposed_transaction_batches: []proto.BatchHash{batchHash},
	}

	b.handler.Emit(ctx, EventNominating{Tip: tip, NumTx: len(txsToNominate), Proposal: proposal})

	// save batches
	b.handler.AddBatch(ctx, batch)

	// disseminate batches; this is non-blocking (no network ops)
	b.handler.Disseminate(ctx, &proto.BatchMessage{Batch_transactions: batch})

	// nominate to consensus
	prevSummary := proto.PrevProposalSummary{
		Block_hash: nom.prevBlockHash,
	}
	// block until nomination is over
	b.handler.Nominate(ctx, nom.blockHeight, proposal, &prevSummary) // ignore result
	b.handler.Emit(ctx, EventNominated{Tip: tip, NumTx: len(txsToNominate), Proposal: proposal})

	return proposal, nom.dbBlockID, nil
}

type nomination struct {
	dbBlockID     proto.BlockID
	blockHeight   proto.BlockHeight
	blockHash     proto.BlockHash
	prevBlockHash proto.BlockHash
	proposedTxs   []proto.ProposedTransaction
	batches       []*proto.BatchTransactions
}

type proposed struct {
	err error
	txs []proto.ProposedTransaction
}

// prepareNomination can be aborted by cancelling the context.
func (b *Baton) prepareNomination(
	ctx context.Context,
	tip Tip,
	txs []*proto.SignedTransaction,
) (*nomination, error) {

	// connect to db
	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	blockID := b.handler.GenerateBlockID(ctx)

	// db create snapshot
	dbResult, err := db.Dbcoord_create_snapshot(ctx, tip.ID, blockID)
	if err != nil {
		return nil, err
	}
	if dbResult != proto.DbResultOK {
		return nil, fmt.Errorf("db create snapshot rpc error")
	}

	// connect to available executors
	execs, err := b.handler.DialExecs(ctx)
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return nil, err
	}
	if len(execs) == 0 {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return nil, ErrNoExecs
	}
	for _, exec := range execs {
		defer exec.Close()
	}

	// propose all transactions
	txCh := make(chan *proto.SignedTransaction, len(txs))
	for _, tx := range txs {
		txCh <- tx
	}
	close(txCh)

	proposedCh := make(chan proposed, len(execs))
	for _, exec := range execs {
		go func(exec ExecCoordConn) {

			// executor open proposal block
			execResult, err := exec.Executorcoord_open_proposal_block(ctx, blockID)
			if err != nil {
				proposedCh <- proposed{err: err}
				return
			}
			if execResult != proto.ExecResultOK {
				proposedCh <- proposed{err: fmt.Errorf("executor open proposal block rpc error")}
				return
			}

			proposedTxs := []proto.ProposedTransaction{}
			for tx := range txCh {
				select {
				case <-ctx.Done():
					proposedCh <- proposed{err: fmt.Errorf("context cancelled")}
				default:
				}

				proposeResult, err := exec.Executorcoord_propose_transaction(ctx, blockID, *tx)
				if err != nil {
					proposedCh <- proposed{err: err}
					return
				}
				if proposeResult.Result != proto.ExecResultOK {
					proposedCh <- proposed{err: fmt.Errorf("executor propose transaction rpc error")}
					return
				}
				proposedTxs = append(
					proposedTxs,
					proto.ProposedTransaction{
						Signed_tx:                      *tx,
						Nondeterministic_proposal_data: *proposeResult.Proposal_data(),
					},
				)
			}

			// executor finish proposal block
			err = exec.Executorcoord_finish_block(ctx, blockID)
			if err != nil {
				proposedCh <- proposed{err: fmt.Errorf("executor finish error: %v", err)}
				return
			}

			proposedCh <- proposed{err: nil, txs: proposedTxs}

		}(exec)
	}
	var proposedTxs []proto.ProposedTransaction
	var verr error
	for range execs {
		if pd := <-proposedCh; pd.err != nil {
			verr = err
		} else {
			proposedTxs = append(proposedTxs, pd.txs...)
		}
	}
	if verr != nil {
		// db abandon snapshot
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return nil, fmt.Errorf("proposal error: %v", verr)
	}

	// db close snapshot
	dbBlockResult, err := db.Dbcoord_close_snapshot(ctx, blockID)
	if err != nil {
		return nil, err
	}
	if dbBlockResult.Result != proto.DbResultOK {
		return nil, fmt.Errorf("db close snapshot rpc error")
	}

	return &nomination{
		dbBlockID:     blockID,
		blockHeight:   tip.Height + 1,
		blockHash:     *dbBlockResult.Block_hash(),
		prevBlockHash: tip.Hash,
		proposedTxs:   proposedTxs,
		batches:       nil, // filled later
	}, nil
}
