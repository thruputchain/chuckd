package testdb

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	proto "github.com/groundhogchain/protocol-go"
)

type TestDb interface {
	GetTip() batonkit.Tip
	RestoreDb(context.Context) (batonkit.Tip, error)
	DbCoord() batonkit.DbCoordConn
	AddProposedTx(ctx context.Context, id proto.BlockID, ptx *proto.ProposedTransaction)
	AddSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction)
	Dbcoord_null(ctx context.Context) error
	Dbcoord_get_tip(context.Context) (proto.DbTipResult, error)
	Dbcoord_open_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error)
	Dbcoord_create_snapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode) (proto.DbResult, error)
	Dbcoord_commit_snapshot(ctx context.Context, id proto.BlockID) (proto.DbBlockResult, error)
	Dbcoord_close_snapshot(ctx context.Context, id proto.BlockID) (proto.DbBlockResult, error)
	Dbcoord_abandon_snapshot(ctx context.Context, id proto.BlockID) (proto.DbResult, error)
	Dbcoord_catchup(context.Context, proto.BlockHeight) (proto.DbCatchupResult, error)
	Dbcoord_get_hog_ages(context.Context, proto.ContractAddressBatch) (proto.HogAgeResult, error)
	Dbcoord_write_hog_ages(context.Context, proto.HogAgeBatch) (proto.DbResult, error)
	Dbcoord_load_genesis_block(context.Context, proto.GenesisBlock) (proto.DbResult, error)
	Dbcoord_shard_restore(ctx context.Context, a1 proto.BlockHeight) (r proto.DbShardResult, _ error)
	Close()
}
