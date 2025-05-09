package baton

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type nominate struct {
	tipHeight proto.BlockHeight
	tipHash   proto.BlockHash
	tipID     proto.BlockID
}

func (b *Baton) startNominating(ctx context.Context) {
	var interruptingTip *nominate
	for {

		var tip *nominate
		if interruptingTip == nil {
			// nominate signals that our db is up to speed, and
			// there are no known externalizations above us
			tip = waitAndDrain(b.nominate)
		} else {
			interruptingTip = nil
			tip = drain(b.nominate, interruptingTip)
		}

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

		// start working on proposing a block, while being ready to cancel if a block is externalized by someone else
		prepCtx, prepCancel := context.WithCancel(ctx)
		prepCh := make(chan *nomination, 1)
		go func() {
			nom, err := b.prepareNomination(prepCtx, tip, txsToNominate)
			if err != nil {
				log.Errorf("preparing nomination: %v", err)
			}
			prepCh <- nom
		}()

		// wait for nomination preparation or for a new tip interruption
		var nom *nomination
	WaitPrepReadyOrInterrupt:
		for {
			select {
			case interruptingTip = <-b.nominate: // a block has been externalized, abort nomination
				// the baton may send a nomination signal pointing at the same tip; it must be ignored
				if interruptingTip.tipHash == tip.tipHash {
					continue // ignore interruptions from the same tip!
				} else {
					break WaitPrepReadyOrInterrupt
				}
			case nom = <-prepCh:
				break WaitPrepReadyOrInterrupt
			}
		}
		prepCancel()
		if nom == nil { // we were interrupted
			continue
		}

		// construct proposed block
		batch := proto.BatchTransactions{
			Transactions: nom.proposedTxs,
		}
		batchHash := proto.BatchHash(sha256.Sum256(xdrutil.MarshalXdr(&batch)))
		proposal := &proto.Proposal{
			Block_height:                 nom.blockHeight,
			Block_hash:                   nom.blockHash,
			Prev_block_hash:              nom.prevBlockHash,
			Proposed_transaction_batches: []proto.BatchHash{batchHash},
		}

		// disseminate batches; this is non-blocking (no network ops)
		b.handler.Disseminate(ctx, &proto.BatchMessage{Batch_transactions: batch})

		// record our latest proposal so we can skip validation on the commit path
		// must be done before nomination begins
		b.nominator.lk.Lock()
		b.nominator.latestBlockID = &nom.dbBlockID
		b.nominator.latestProposal = proposal
		b.nominator.lk.Unlock()

		// nominate to consensus
		nomCtx, nomCancel := context.WithCancel(ctx)
		nomCh := make(chan bool, 1)
		go func() {
			prevSummary := proto.PrevProposalSummary{
				Block_hash: nom.prevBlockHash,
			}
			nomCh <- b.handler.Nominate(nomCtx, nom.blockHeight, proposal, &prevSummary)
		}()

		var outcome *bool
		select {
		case interruptingTip = <-b.nominate: // a block has been externalized, abort nomination
		case r := <-nomCh:
			outcome = &r
		}
		nomCancel()
		if outcome == nil {
			break
		}
	}
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
	tip *nominate,
	txs []*proto.SignedTransaction,
) (*nomination, error) {

	// connect to db
	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	nextID := b.handler.GenerateBlockID(ctx)

	// db open snapshot
	dbResult, err := db.Dbcoord_open_snapshot(ctx, tip.tipID, nextID)
	if err != nil {
		return nil, err
	}
	if dbResult != proto.DbResultOK {
		return nil, fmt.Errorf("db open snapshot rpc error")
	}

	// connect to available executors
	execs, err := b.handler.DialExecs(ctx)
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, nextID)
		return nil, err
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
			execResult, err := exec.Executorcoord_open_proposal_block(ctx, nextID)
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

				proposeResult, err := exec.Executorcoord_propose_transaction(ctx, nextID, *tx)
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
			err = exec.Executorcoord_finish_block(ctx, nextID)
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
		db.Dbcoord_abandon_snapshot(ctx, nextID)
		return nil, fmt.Errorf("proposal error: %v", verr)
	}

	// db commit snapshot
	dbBlockResult, err := db.Dbcoord_commit_snapshot(ctx, nextID)
	if err != nil {
		return nil, err
	}
	if dbBlockResult.Result != proto.DbResultOK {
		return nil, fmt.Errorf("db commit snapshot rpc error")
	}

	return &nomination{
		dbBlockID:     nextID,
		blockHeight:   tip.tipHeight + 1,
		blockHash:     *dbBlockResult.Block_hash(),
		prevBlockHash: tip.tipHash,
		proposedTxs:   proposedTxs,
		batches:       nil, // filled later
	}, nil
}
