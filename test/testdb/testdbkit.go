package testdb

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type TestDb interface {
	GetTip() proto.DbTip
	RestoreDb(context.Context) (proto.DbTip, error)
	AddProposedTx(ctx context.Context, id proto.BlockID, ptx *proto.ProposedTransaction)
	AddSignedTx(ctx context.Context, id proto.BlockID, stx *proto.SignedTransaction)
	OpenSnapshot(ctx context.Context, id proto.BlockID)
	CreateSnapshot(ctx context.Context, parentID proto.BlockID, id proto.BlockID, mode proto.BlockExecutionMode)
	CommitSnapshot(ctx context.Context, id proto.BlockID) proto.BlockHash
	CloseSnapshot(ctx context.Context, id proto.BlockID) proto.BlockHash
	AbandonSnapshot(ctx context.Context, id proto.BlockID)
	GetCatchup(context.Context, proto.BlockHeight) *proto.CatchupBlock
}
