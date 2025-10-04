package loadgen

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/coord"
	proto "github.com/groundhogchain/protocol-go"
)

type LiveHandler struct {
	coord *coord.CoordinatorService
	rlk   sync.Mutex
	rand  *rand.Rand
}

func NewLiveHandler(
	coord *coord.CoordinatorService,
	rand *rand.Rand,
) *LiveHandler {
	return &LiveHandler{
		coord: coord,
		rand:  rand,
	}
}

func (x *LiveHandler) Committed(context.Context, *proto.CatchupBlock) {
}

func (x *LiveHandler) GenerateBlockID(context.Context) proto.BlockID {
	x.rlk.Lock()
	defer x.rlk.Unlock()
	return x.rand.Uint64()
}

func (x *LiveHandler) DialDb(ctx context.Context) (batonapi.DbCoordConn, error) {
	return x.coord.DialDb(ctx)
}

func (x *LiveHandler) DialExecs(ctx context.Context) ([]batonapi.ExecCoordConn, error) {
	return x.coord.DialExecs(ctx)
}

func (x *LiveHandler) Emit(ctx context.Context, event any) {
}

// submitted mempool

func (x *LiveHandler) TxCount() int {
	panic("not used")
}

func (x *LiveHandler) ListTxs(context.Context) []*proto.SignedTransaction {
	panic("not used")
}

func (x *LiveHandler) PartialLookupTxs(context.Context, []proto.SignedTransactionHash) []*proto.SignedTransaction {
	panic("not used")
}

func (x *LiveHandler) LookupTxs(context.Context, []proto.SignedTransactionHash) ([]*proto.SignedTransaction, bool) {
	panic("not used")
}

func (x *LiveHandler) WaitTxs(context.Context, []proto.SignedTransactionHash) ([]*proto.SignedTransaction, bool) {
	panic("not used")
}

func (x *LiveHandler) RemoveTx(context.Context, proto.SignedTransactionHash) {
}

// committed mempool

func (x *LiveHandler) CommittedTxCount() int {
	panic("not used")
}

func (x *LiveHandler) AddCommittedTx(context.Context, proto.SignedTransactionHash) {
}

func (x *LiveHandler) IsCommittedTx(context.Context, proto.SignedTransactionHash) bool {
	panic("not used")
}
