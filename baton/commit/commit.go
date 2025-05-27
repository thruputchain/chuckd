package commit

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func CommitAlreadyValidated(ctx context.Context, handler sdk.VPCHandler, vb *sdk.ValidatableBlock) (proto.BlockID, error) {
	liblog.Debugf("committing already validated [%d; %s]", vb.BlockHeight, form.Display(vb.BlockHash))

	// connect to db
	db, err := handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := *vb.AlreadyValidatedBlockID

	// db open snapshot
	dbResult, err := db.Dbcoord_open_snapshot(ctx, blockID)
	if err != nil {
		return 0, err
	}
	if dbResult != proto.DbResultOK {
		return 0, fmt.Errorf("db open snapshot error")
	}

	return FinalizeCommit(ctx, handler, vb, blockID, db)
}

func FinalizeCommit(
	ctx context.Context,
	handler sdk.VPCHandler,
	vb *sdk.ValidatableBlock,
	blockID proto.BlockID,
	db sdk.DbCoordConn,
) (proto.BlockID, error) {

	// db commit snapshot
	dbBlockResult, err := db.Dbcoord_commit_snapshot(ctx, blockID)
	if err != nil {
		return 0, err
	}
	if dbBlockResult.Result != proto.DbResultOK {
		return 0, fmt.Errorf("db commit snapshot rpc error")
	}

	// verify the block hash returned by the db matches
	must.Assertf(ctx, *dbBlockResult.Block_hash() == vb.BlockHash,
		"db block hash changed after a prior proposal or validation")

	// record committed transactions to prevent replay
	for _, ptx := range vb.ProposedTransactions {
		hash := sha256.Sum256(xdrutil.MarshalXdr(&ptx.Signed_tx))
		handler.AddCommittedTx(ctx, hash)
		handler.RemoveTx(ctx, hash)
	}

	handler.Committed(ctx, &proto.CatchupBlock{
		Block_height:          vb.BlockHeight,
		Block_hash:            vb.BlockHash,
		Prev_block_hash:       vb.PrevBlockHash,
		Proposed_transactions: protoutil.PackProposedTransactions(vb.ProposedTransactions),
	})
	handler.Emit(ctx, sdk.EventCommitted{ValidatableBlock: vb, BlockID: blockID})

	return blockID, nil
}
