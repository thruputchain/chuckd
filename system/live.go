package system

import (
	"context"

	"github.com/groundhogchain/chuckd/baton"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

// LiveHandler is a baton.Handler that connects a live system to the baton.
type LiveHandler struct {
	protoutil.Signer
	libtime.Time
	*mempool.BatchMempool
	*mempool.TransactionMempool
	*mempool.CommittedMempool
}

func (x *LiveHandler) Disseminate(ctx context.Context, value xdr.XdrType) {
	XXX
}

func (x *LiveHandler) Nominate(ctx context.Context, slot int64, value xdr.XdrType, prev xdr.XdrType) bool {
	XXX
}

func (x *LiveHandler) GetCatchupSources(context.Context) []proto.ValidatorPublicInfo {
	XXX
}

func (x *LiveHandler) RestoreDb(context.Context) (baton.Tip, error) {
	XXX
}

func (x *LiveHandler) GenerateBlockID(context.Context) proto.BlockID {
	XXX
}

func (x *LiveHandler) GenerateID64(context.Context) baton.ID64 {
	XXX
}

func (x *LiveHandler) DialCatchup(context.Context, proto.ValidatorPublicInfo) (baton.CatchupConn, error) {
	XXX
}

func (x *LiveHandler) DialDb(context.Context) (baton.DbCoordConn, error) {
	XXX
}

func (x *LiveHandler) DialExecs(context.Context) ([]baton.ExecCoordConn, error) {
	XXX
}

func (x *LiveHandler) PrioritizeRejectTxs(context.Context, []*proto.SignedTransaction) ([]*proto.SignedTransaction, []*proto.RejectedTransaction) {
	XXX
}

func (x *LiveHandler) RejectedTx(context.Context, *proto.RejectedTransaction) {
	XXX
}

func (x *LiveHandler) Committed(context.Context, *proto.CatchupBlock) {
	XXX
}

func (x *LiveHandler) Emit(ctx context.Context, event any) {
	XXX
}
