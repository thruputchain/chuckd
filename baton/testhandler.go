package baton

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type TestHandler struct {
	signer proto.ValidatorPrivateInfo
	//
	*libtime.TestTime
	*mempool.BatchMempool
	*mempool.TransactionMempool
	*mempool.CommittedMempool
	//
	dlk          sync.Mutex
	disseminated []xdr.XdrType
	//
	nominateIn  chan nominateIn
	nominateOut chan bool
}

func NewTestHandler(
	ctx context.Context,
	now time.Time,
	signer proto.ValidatorPrivateInfo,
	batchTTL time.Duration,
	txTTL time.Duration,
	committedTTL time.Duration,
) *TestHandler {
	time := libtime.NewTestTime(now)
	return &TestHandler{
		signer:             signer,
		TestTime:           time,
		BatchMempool:       mempool.NewBatchMempool(ctx, time, batchTTL),
		TransactionMempool: mempool.NewTransactionMempool(ctx, time, txTTL),
		CommittedMempool:   mempool.NewCommittedMempool(ctx, time, committedTTL),
		nominateIn:         make(chan nominateIn),
		nominateOut:        make(chan bool),
	}
}

func (h *TestHandler) Disseminate(ctx context.Context, value xdr.XdrType) {
	h.dlk.Lock()
	defer h.dlk.Unlock()
	h.disseminated = append(h.disseminated, value)
}

type nominateIn struct {
	Slot  int64
	Value xdr.XdrType
	Prev  xdr.XdrType
}

func (h *TestHandler) Nominate(ctx context.Context, slot int64, value xdr.XdrType, prev xdr.XdrType) bool {
	h.nominateIn <- nominateIn{Slot: slot, Value: value, Prev: prev}
	return <-h.nominateOut
}

func (h *TestHandler) NextNomination() (slot int64, value xdr.XdrType, prev xdr.XdrType) {
	x := <-h.nominateIn
	return x.Slot, x.Value, x.Prev
}

func (h *TestHandler) SignalNomination(result bool) {
	h.nominateOut <- result
}

func (h *TestHandler) Signer() proto.ValidatorPublicInfo {
	return h.signer.Public_info
}

func (h *TestHandler) SignXdr(v xdr.XdrType) proto.Signature {
	return protoutil.SignXdr(h.signer.Secret_key, v)
}

func (h *TestHandler) VerifyXdr(signer proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool {
	return protoutil.VerifyXdr(signer, value, sig)
}
