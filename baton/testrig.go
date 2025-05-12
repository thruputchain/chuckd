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
	dlk          sync.Mutex
	disseminated []xdr.XdrType
	//
	nominateIn  chan nominateIn
	nominateOut chan bool
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
		nominateIn:         make(chan nominateIn),
		nominateOut:        make(chan bool),
	}
	h.TestDb = NewTestDb(0, proto.BlockHash{}, 0)
	catchupSigner, err := protoutil.GenerateIdentity(rand, "test")
	must.NoError(ctx, err)
	h.TestCatchup = NewTestCatchup(*catchupSigner)
	return h
}

func (h *TestRig) Bind(b *Baton) {
	h.TestSCP = NewTestSCP(b.SCPHandler())
}

func (h *TestRig) Emit(ctx context.Context, event any) {
	h.TestBus.SendEvent(h.TestTime.Now(), event)
}

func (h *TestRig) Disseminate(ctx context.Context, value xdr.XdrType) {
	h.dlk.Lock()
	defer h.dlk.Unlock()
	h.disseminated = append(h.disseminated, value)
}

type nominateIn struct {
	Slot  int64
	Value xdr.XdrType
	Prev  xdr.XdrType
}

func (h *TestRig) Nominate(ctx context.Context, slot int64, value xdr.XdrType, prev xdr.XdrType) bool {
	h.nominateIn <- nominateIn{Slot: slot, Value: value, Prev: prev}
	return <-h.nominateOut
}

func (h *TestRig) Control_WaitNextNomination(ctx context.Context) (slot int64, value xdr.XdrType, prev xdr.XdrType) {
	x := <-h.nominateIn
	return x.Slot, x.Value, x.Prev
}

func (h *TestRig) Control_GetNextNomination(ctx context.Context) (slot int64, value xdr.XdrType, prev xdr.XdrType) {
	select {
	case x := <-h.nominateIn:
		return x.Slot, x.Value, x.Prev
	default:
		must.Assertf(ctx, false, "no nomination available")
	}
	panic("unreachable")
}

func (h *TestRig) SignalNomination(result bool) {
	h.nominateOut <- result
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
	must.Assertf(ctx, len(h.nextExecs) > 0, "no staged execs")
	execs := h.nextExecs[0]
	h.nextExecs = h.nextExecs[1:]
	return execs, nil
}

func (h *TestRig) PrioritizeTxs(ctx context.Context, txs []*proto.SignedTransaction) []*proto.SignedTransaction {
	return txs
}
