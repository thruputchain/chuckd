package baton

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/baton/validate"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	proto "github.com/groundhogchain/protocol-go"
)

type validated struct {
	*proto.Proposal
	proto.BlockID
}

func (b *Baton) validateCommit(ctx context.Context, vb *sdk.ValidatableBlock) (proto.BlockID, error) {
	liblog.Debugf("validating and committing [%d; %s]", vb.BlockHeight, form.Display(vb.BlockHash))

	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := b.handler.GenerateBlockID(ctx)

	if err = validate.Validate(ctx, b.handler, vb, db, blockID); err != nil {
		return 0, err
	}

	return b.finalizeCommit(ctx, vb, blockID, db)
}

func (b *Baton) validateClose(ctx context.Context, vb *sdk.ValidatableBlock) (proto.BlockID, error) {

	db, err := b.handler.DialDb(ctx)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	blockID := b.handler.GenerateBlockID(ctx)

	if err = validate.Validate(ctx, b.handler, vb, db, blockID); err != nil {
		return 0, err
	}

	return blockID, b.finalizeValidateClose(ctx, vb, blockID, db)
}

func (b *Baton) finalizeValidateClose(
	ctx context.Context,
	vb *sdk.ValidatableBlock,
	blockID proto.BlockID,
	db sdk.DbCoordConn,
) error {

	// db close snapshot
	dbBlockResult, err := db.Dbcoord_close_snapshot(ctx, blockID)
	if err != nil {
		return err
	}
	if dbBlockResult.Result != proto.DbResultOK {
		return fmt.Errorf("db commit snapshot rpc error")
	}

	// verify the block hash returned by the db matches
	if *dbBlockResult.Block_hash() != vb.BlockHash {
		liblog.Infof("cannot validate, resulting block hash %v != proposed block hash %v",
			form.Display(*dbBlockResult.Block_hash()), form.Display(vb.BlockHash))
		return sdk.ErrNotValid
	}

	return nil
}
