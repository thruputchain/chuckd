package nominate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/sdk"
	proto "github.com/groundhogchain/protocol-go"
)

type Handler interface {
	GenerateBlockID(context.Context) proto.BlockID
	DialDb(context.Context) (sdk.DbCoordConn, error)
	DialExecs(context.Context) ([]sdk.ExecCoordConn, error)
}

type Nomination struct {
	BlockID       proto.BlockID
	BlockHeight   proto.BlockHeight
	BlockHash     proto.BlockHash
	PrevBlockHash proto.BlockHash
	ProposedTxs   []proto.ProposedTransaction
	Batches       []*proto.BatchTransactions
}

type proposed struct {
	err error
	txs []proto.ProposedTransaction
}

// PrepareNomination can be aborted by cancelling the context.
func PrepareNomination(
	ctx context.Context,
	handler Handler,
	tip sdk.Tip,
	txs []*proto.SignedTransaction,
) (*Nomination, error) {

	// connect to db
	db, err := handler.DialDb(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	blockID := handler.GenerateBlockID(ctx)

	// db create snapshot
	dbResult, err := db.Dbcoord_create_snapshot(ctx, tip.ID, blockID, proto.KBlockExecutionModeProposal)
	if err != nil {
		return nil, err
	}
	if dbResult != proto.DbResultOK {
		return nil, fmt.Errorf("db create snapshot rpc error")
	}

	// connect to available executors
	execs, err := handler.DialExecs(ctx)
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return nil, err
	}
	if len(execs) == 0 {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return nil, sdk.ErrNoExecs
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
		go func(exec sdk.ExecCoordConn) {

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

	return &Nomination{
		BlockID:       blockID,
		BlockHeight:   tip.Height + 1,
		BlockHash:     *dbBlockResult.Block_hash(),
		PrevBlockHash: tip.Hash,
		ProposedTxs:   proposedTxs,
		Batches:       nil, // filled later
	}, nil
}
