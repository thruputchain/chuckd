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

type TestRig struct {
	user   proto.ValidatorSecretKey
	signer proto.ValidatorPrivateInfo
	*TestBus
	*libtime.TestTime
	*mempool.BatchMempool
	*mempool.TransactionMempool
	*mempool.CommittedMempool
	*TestDb
	*TestCatchup
	*TestSCP
	//
	xlk       sync.Mutex
	nextExecs [][]ExecCoordConn
	//
	rlk  sync.Mutex
	rand *rand.Rand
	//
	disseminated chan xdr.XdrType
	//
	nominateRequest  chan nominateIn
	nominateResponse chan bool
}

func NewTestRig(
	ctx context.Context,
	rand *rand.Rand,
	now time.Time,
	batchTTL time.Duration,
	txTTL time.Duration,
	committedTTL time.Duration,
) *TestRig {
	user, err := protoutil.GenerateIdentity(rand, "test_user")
	must.NoError(ctx, err)
	signer, err := protoutil.GenerateIdentity(rand, "test_validator")
	must.NoError(ctx, err)
	time := libtime.NewTestTime(now)
	h := &TestRig{
		rand:               rand,
		user:               user.Secret_key,
		signer:             *signer,
		TestBus:            NewTestBus(),
		TestTime:           time,
		BatchMempool:       mempool.NewBatchMempool(ctx, time, batchTTL),
		TransactionMempool: mempool.NewTransactionMempool(ctx, time, txTTL),
		CommittedMempool:   mempool.NewCommittedMempool(ctx, time, committedTTL),
		disseminated:       make(chan xdr.XdrType),
		nominateRequest:    make(chan nominateIn, 1),
		nominateResponse:   make(chan bool, 1),
	}
	h.TestDb = NewTestDb(Tip{})
	catchupSigner, err := protoutil.GenerateIdentity(rand, "test")
	must.NoError(ctx, err)
	h.TestCatchup = NewTestCatchup(time, *catchupSigner)
	return h
}

func (h *TestRig) Bind(b *Baton) {
	h.TestSCP = NewTestSCP(b.SCPHandler())
}

func (h *TestRig) Emit(ctx context.Context, event any) {
	h.TestBus.SendEvent(h.TestTime.Now(), event)
}

func (h *TestRig) Disseminate(ctx context.Context, value xdr.XdrType) {
	h.disseminated <- value
}

func (h *TestRig) WaitDissemination() xdr.XdrType {
	return <-h.disseminated
}

type nominateIn struct {
	Slot  int64
	Value xdr.XdrType
	Prev  xdr.XdrType
}

// Nominate is called by Baton
func (h *TestRig) Nominate(ctx context.Context, slot int64, value xdr.XdrType, prev xdr.XdrType) bool {
	h.nominateRequest <- nominateIn{Slot: slot, Value: value, Prev: prev}
	return <-h.nominateResponse
}

// WaitNomination waits until Baton calls Nominate
func (h *TestRig) WaitNomination(ctx context.Context) (slot int64, value xdr.XdrType, prev xdr.XdrType) {
	x := <-h.nominateRequest
	return x.Slot, x.Value, x.Prev
}

func (h *TestRig) SignalNomination(result bool) {
	h.nominateResponse <- result
}

func (h *TestRig) Signer() proto.ValidatorPublicInfo {
	return h.signer.Public_info
}

func (h *TestRig) SignXdr(v xdr.XdrType) proto.Signature {
	return protoutil.SignXdr(h.signer.Secret_key, v)
}

func (h *TestRig) VerifyXdr(signer proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool {
	return protoutil.VerifyXdr(signer, value, sig)
}

func (h *TestRig) GetCatchupSources(context.Context) []proto.ValidatorPublicInfo {
	return []proto.ValidatorPublicInfo{h.TestCatchup.Signer.Public_info}
}

func (h *TestRig) GenerateBlockID(context.Context) proto.BlockID {
	h.rlk.Lock()
	defer h.rlk.Unlock()
	return h.rand.Uint64()
}

func (h *TestRig) GenerateID64(context.Context) ID64 {
	h.rlk.Lock()
	defer h.rlk.Unlock()
	return ID64(h.rand.Int63())
}

func (h *TestRig) DialCatchup(context.Context, proto.ValidatorPublicInfo) (CatchupConn, error) {
	return h.TestCatchup, nil
}

func (h *TestRig) DialDb(context.Context) (DbCoordConn, error) {
	return h.TestDb, nil
}

func (h *TestRig) StageExecs(n int) []*TestExec {
	testExecs := make([]*TestExec, n)
	execs := make([]ExecCoordConn, n)
	for i := range testExecs {
		testExecs[i] = NewTestExec(h.TestDb)
		execs[i] = testExecs[i]
	}
	h.xlk.Lock()
	defer h.xlk.Unlock()
	h.nextExecs = append(h.nextExecs, execs)
	return testExecs
}

func (h *TestRig) DialExecs(ctx context.Context) ([]ExecCoordConn, error) {
	h.xlk.Lock()
	defer h.xlk.Unlock()
	// if there are no staged execs, just allocate
	if len(h.nextExecs) > 0 {
		execs := h.nextExecs[0]
		h.nextExecs = h.nextExecs[1:]
		return execs, nil
	} else {
		return []ExecCoordConn{NewTestExec(h.TestDb), NewTestExec(h.TestDb)}, nil
	}
}

func (h *TestRig) PrioritizeRejectTxs(ctx context.Context, txs []*proto.SignedTransaction) ([]*proto.SignedTransaction, []*proto.RejectedTransaction) {
	return txs, nil
}

func (h *TestRig) RejectedTx(context.Context, *proto.RejectedTransaction) {}

func (h *TestRig) Committed(context.Context, *proto.CatchupBlock) {}
