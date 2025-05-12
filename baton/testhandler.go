package baton

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type TestHandler struct {
	signer proto.ValidatorPrivateInfo
	*libtime.TestTime
	*mempool.BatchMempool
	*mempool.TransactionMempool
	*mempool.CommittedMempool
	*TestDb
	*TestCatchup
	//
	xlk       sync.Mutex
	nextExecs []ExecCoordConn
	//
	rlk  sync.Mutex
	rand *rand.Rand
	//
	dlk          sync.Mutex
	disseminated []xdr.XdrType
	//
	nominateIn  chan nominateIn
	nominateOut chan bool
}

func NewTestHandler(
	ctx context.Context,
	rand *rand.Rand,
	//
	now time.Time,
	signer proto.ValidatorPrivateInfo,
	//
	batchTTL time.Duration,
	txTTL time.Duration,
	committedTTL time.Duration,
	//
	tipHeight proto.BlockHeight,
	tipHash proto.BlockHash,
	tipID proto.BlockID,
) *TestHandler {
	time := libtime.NewTestTime(now)
	h := &TestHandler{
		rand:               rand,
		signer:             signer,
		TestTime:           time,
		BatchMempool:       mempool.NewBatchMempool(ctx, time, batchTTL),
		TransactionMempool: mempool.NewTransactionMempool(ctx, time, txTTL),
		CommittedMempool:   mempool.NewCommittedMempool(ctx, time, committedTTL),
		nominateIn:         make(chan nominateIn),
		nominateOut:        make(chan bool),
	}
	h.TestDb = NewTestDb(tipHeight, tipHash, tipID)
	catchupSigner, err := protoutil.GenerateIdentity(rand, "test")
	must.NoError(ctx, err)
	h.TestCatchup = NewTestCatchup(*catchupSigner)
	return h
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

func (h *TestHandler) Control_WaitNextNomination(ctx context.Context) (slot int64, value xdr.XdrType, prev xdr.XdrType) {
	x := <-h.nominateIn
	return x.Slot, x.Value, x.Prev
}

func (h *TestHandler) Control_GetNextNomination(ctx context.Context) (slot int64, value xdr.XdrType, prev xdr.XdrType) {
	select {
	case x := <-h.nominateIn:
		return x.Slot, x.Value, x.Prev
	default:
		must.Assertf(ctx, false, "no nomination available")
	}
	panic("unreachable")
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

func (h *TestHandler) GetCatchupSources(context.Context) []proto.ValidatorPublicInfo {
	return []proto.ValidatorPublicInfo{h.TestCatchup.Signer.Public_info}
}

func (h *TestHandler) GenerateBlockID(context.Context) proto.BlockID {
	h.rlk.Lock()
	defer h.rlk.Unlock()
	return h.rand.Uint64()
}

func (h *TestHandler) GenerateID64(context.Context) ID64 {
	h.rlk.Lock()
	defer h.rlk.Unlock()
	return ID64(h.rand.Int63())
}

func (h *TestHandler) DialCatchup(context.Context, proto.ValidatorPublicInfo) (CatchupConn, error) {
	return h.TestCatchup, nil
}

func (h *TestHandler) DialDb(context.Context) (DbCoordConn, error) {
	return h.TestDb, nil
}

func (h *TestHandler) Control_PrepNextExecs(n int) []*TestExec {
	testExecs := make([]*TestExec, n)
	execs := make([]ExecCoordConn, n)
	for i := range testExecs {
		testExecs[i] = NewTestExec(h.TestDb)
		execs[i] = testExecs[i]
	}
	h.xlk.Lock()
	defer h.xlk.Unlock()
	h.nextExecs = execs
	return testExecs
}

func (h *TestHandler) DialExecs(context.Context) ([]ExecCoordConn, error) {
	h.xlk.Lock()
	defer h.xlk.Unlock()
	execs := h.nextExecs
	h.nextExecs = nil
	return execs, nil
}

func (h *TestHandler) PrioritizeTxs(ctx context.Context, txs []*proto.SignedTransaction) []*proto.SignedTransaction {
	return txs
}
