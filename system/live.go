package system

import (
	"context"

	"github.com/groundhogchain/chuckd/baton"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/net/multiplex"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

// LiveHandler is a baton.Handler that connects a live system to the baton.
type LiveHandler struct {
	protoutil.Signer
	libtime.Time
	router router.Router
	*mempool.BatchMempool
	*mempool.TransactionMempool
	*mempool.CommittedMempool
	multiplex *multiplex.Multiplex
	scp       api.SCPController
	//
	live chan struct{}
}

func NewLiveHandler(
	signer protoutil.Signer,
	time libtime.Time,
	router router.Router,
	batch *mempool.BatchMempool,
	tx *mempool.TransactionMempool,
	committed *mempool.CommittedMempool,
	multiplex *multiplex.Multiplex,
) *LiveHandler {
	return &LiveHandler{
		Signer:             signer,
		Time:               time,
		router:             router,
		BatchMempool:       batch,
		TransactionMempool: tx,
		CommittedMempool:   committed,
		multiplex:          multiplex,
		live:               make(chan struct{}),
	}
}

func (x *LiveHandler) SetSCP(scp api.SCPController) {
	x.scp = scp
	close(x.live)
}

func (x *LiveHandler) Disseminate(ctx context.Context, value *proto.BatchMessage) {
	<-x.live
	x.multiplex.SendBatch(ctx, value)
}

func (x *LiveHandler) Nominate(ctx context.Context, slot int64, value xdr.XdrType, prev xdr.XdrType) bool {
	<-x.live
	return x.scp.Nominate(ctx, slot, xdrutil.MarshalXdr(value), xdrutil.MarshalXdr(prev))
}

func (x *LiveHandler) GetCatchupSources(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	<-x.live
	return x.router.ChooseCatchupProviders(ctx, seed)
}

func (x *LiveHandler) RestoreDb(context.Context) (baton.Tip, error) {
	<-x.live
	XXX
}

func (x *LiveHandler) GenerateBlockID(context.Context) proto.BlockID {
	XXX
}

func (x *LiveHandler) GenerateID64(context.Context) baton.ID64 {
	XXX
}

func (x *LiveHandler) DialCatchup(context.Context, proto.ValidatorPublicInfo) (baton.CatchupConn, error) {
	<-x.live
	XXX
}

func (x *LiveHandler) DialDb(context.Context) (baton.DbCoordConn, error) {
	<-x.live
	XXX
}

func (x *LiveHandler) DialExecs(context.Context) ([]baton.ExecCoordConn, error) {
	<-x.live
	XXX
}

func (x *LiveHandler) PrioritizeRejectTxs(_ context.Context, txs []*proto.SignedTransaction) ([]*proto.SignedTransaction, []*proto.RejectedTransaction) {
	return txs, nil
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
