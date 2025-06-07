package nominate

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/exec1"
	proto "github.com/groundhogchain/protocol-go"
)

type Nomination struct {
	BlockID       proto.BlockID
	BlockHeight   proto.BlockHeight
	BlockHash     proto.BlockHash
	PrevBlockHash proto.BlockHash
	ProposedTxs   []ProposedTx
}

func (x Nomination) ProposedTransactions() []proto.ProposedTransaction {
	ptxs := []proto.ProposedTransaction{}
	for _, ptx := range x.ProposedTxs {
		if ptx.Err != nil {
			continue
		}
		ptxs = append(ptxs, proto.ProposedTransaction{
			Signed_tx:                      *ptx.Tx,
			Nondeterministic_proposal_data: *ptx.Result,
		})
	}
	return ptxs
}

// nomination calls each executor on callsPerExec concurrent threads.
// this produces a similar effect as join-bounded-shortest-queue (JBSQ)
// in the case that there always are rpc calls ready to be fired.
// see https://www.usenix.org/conference/atc19/presentation/kogias-r2p2 JBSQ
// I would make the bound (callsPerExec) 2*N where N is the number of cores used by the executor.

// PrepareNomination can be aborted by cancelling the context.
func PrepareNomination(
	ctx context.Context,
	handler batonapi.OpsHandler,
	tip batonapi.Tip,
	txs []*proto.SignedTransaction,
	callsPerExec int,
	batchSize int,
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

	var proposedTxs []ProposedTx
	if batchSize < 2 {
		proposedTxs, err = execProposeTx(ctx, handler, txs, blockID, callsPerExec)
	} else {
		proposedTxs, err = execProposeBatch(ctx, handler, txs, blockID, callsPerExec, batchSize)
	}
	if err != nil {
		db.Dbcoord_abandon_snapshot(ctx, blockID)
		return nil, fmt.Errorf("proposal error: %v", err)
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
	}, nil
}

func execProposeTx(
	ctx context.Context,
	handler batonapi.OpsHandler,
	txs []*proto.SignedTransaction,
	blockID proto.BlockID,
	callsPerExec int,
) ([]exec1.DoneTx[*proto.SignedTransaction, *proto.NondeterministicProposalData], error) {
	return exec1.Exec(
		ctx,
		handler,
		txs,
		blockID,
		submitProposeTx{},
		callsPerExec,
	)
}

func execProposeBatch(
	ctx context.Context,
	handler batonapi.OpsHandler,
	txs []*proto.SignedTransaction,
	blockID proto.BlockID,
	callsPerExec int,
	batchSize int,
) ([]exec1.DoneTx[*proto.SignedTransaction, *proto.NondeterministicProposalData], error) {
	return exec1.Exec(
		ctx,
		handler,
		txs,
		blockID,
		submitProposeBatch{BatchSize: batchSize},
		callsPerExec,
	)
}
