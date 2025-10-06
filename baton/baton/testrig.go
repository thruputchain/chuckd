package baton

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/test/testcatchup"
	"github.com/groundhogchain/chuckd/test/testdb"
	"github.com/groundhogchain/chuckd/test/testexec"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type TestRig struct {
	user   proto.ValidatorSecretKey
	signer proto.ValidatorPrivateInfo
	*testutil.TestGen
	*TestBus
	*libtime.TestTime
	*mempool.SubmittedMempool
	*mempool.CommittedMempool
	testdb.TestDb
	*testcatchup.TestCatchupStaged
	*TestSCP
	//
	xlk       sync.Mutex
	nextExecs [][]batonapi.ExecCoordConn
	//
	rlk  sync.Mutex
	rand *rand.Rand
	//
	nominateRequest  chan nominateIn
	nominateResponse chan bool
}

func NewTestRig(
	ctx context.Context,
	rand *rand.Rand,
	now time.Time,
	txTTL time.Duration,
	committedTTL time.Duration,
) *TestRig {
	user, err := protoutil.GenerateIdentity(rand, "test_user")
	must.NoError(ctx, err)
	signer, err := protoutil.GenerateIdentity(rand, "test_validator")
	must.NoError(ctx, err)
	time := libtime.NewTestTime(now)
	h := &TestRig{
		rand:   rand,
		user:   user.Secret_key,
		signer: *signer,
		TestGen: testutil.NewTestGen(
			ctx,
			time,
			runtime.NewRandFromRand(rand),
			*signer,
			user.Secret_key,
		),
		TestBus:          NewTestBus(),
		TestTime:         time,
		SubmittedMempool: mempool.NewTransactionMempool(ctx, time, txTTL),
		CommittedMempool: mempool.NewCommittedMempool(ctx, time, committedTTL),
		nominateRequest:  make(chan nominateIn, 1),
		nominateResponse: make(chan bool, 1),
	}
	h.TestDb = testdb.NewTestDbConsistent(batonapi.Tip{})
	catchupSigner, err := protoutil.GenerateIdentity(rand, "test")
	must.NoError(ctx, err)
	h.TestCatchupStaged = testcatchup.NewTestCatchup(time, *catchupSigner)
	return h
}

func (h *TestRig) GenNomination(ctx context.Context, ntxs int) *testutil.TestBlock {
	return h.GenNominationOntoTip(ctx, ntxs, h.TestDb.GetTip())
}

func (h *TestRig) Bind(b *Baton) {
	h.TestSCP = NewTestSCP(b.SCPHandler())
}

func (h *TestRig) Emit(ctx context.Context, event any) {
	h.TestBus.SendEvent(h.TestTime.Now(), event)
}

type nominateIn struct {
	Slot  scpapi.Slot
	Value xdr.XdrType
	Prev  xdr.XdrType
}

// Nominate is called by Baton
func (h *TestRig) Nominate(ctx context.Context, slot scpapi.Slot, value xdr.XdrType, prev xdr.XdrType) bool {
	h.nominateRequest <- nominateIn{Slot: slot, Value: value, Prev: prev}
	return <-h.nominateResponse
}

// WaitNomination waits until Baton calls Nominate
func (h *TestRig) WaitNomination(ctx context.Context) (slot scpapi.Slot, value xdr.XdrType, prev xdr.XdrType) {
	x := <-h.nominateRequest
	return x.Slot, x.Value, x.Prev
}

func (h *TestRig) SignalNomination(result bool) {
	h.nominateResponse <- result
}

func (h *TestRig) Signer() proto.ValidatorPublicInfo {
	return h.signer.Public_info
}

func (h *TestRig) SignXdr(m xdrutil.XdrMarshaler, v xdr.XdrType) proto.Signature {
	return protoutil.SignXdr(m, h.signer.Secret_key, v)
}

func (h *TestRig) VerifyXdr(m xdrutil.XdrMarshaler, signer proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool {
	return protoutil.VerifyXdr(m, signer, value, sig)
}

func (h *TestRig) GetCatchupSources(context.Context, int64) []proto.ValidatorPublicInfo {
	return []proto.ValidatorPublicInfo{h.TestCatchupStaged.Signer.Public_info}
}

func (h *TestRig) GenerateBlockID(context.Context) proto.BlockID {
	h.rlk.Lock()
	defer h.rlk.Unlock()
	return h.rand.Uint64()
}

func (h *TestRig) GenerateID64(context.Context) batonapi.ID64 {
	h.rlk.Lock()
	defer h.rlk.Unlock()
	return batonapi.ID64(h.rand.Int63())
}

func (h *TestRig) DialCatchup(context.Context, proto.ValidatorPublicInfo) (batonapi.CatchupConn, error) {
	return h.TestCatchupStaged, nil
}

func (h *TestRig) DialDb(context.Context) (batonapi.DbCoordConn, error) {
	return h.TestDb, nil
}

func (h *TestRig) StageExecs(n int) []*testexec.TestExec {
	testExecs := make([]*testexec.TestExec, n)
	execs := make([]batonapi.ExecCoordConn, n)
	for i := range testExecs {
		testExecs[i] = testexec.NewTestExec(h.TestDb)
		execs[i] = testExecs[i]
	}
	h.xlk.Lock()
	defer h.xlk.Unlock()
	h.nextExecs = append(h.nextExecs, execs)
	return testExecs
}

func (h *TestRig) DialExecs(ctx context.Context) ([]batonapi.ExecCoordConn, error) {
	h.xlk.Lock()
	defer h.xlk.Unlock()
	// if there are no staged execs, just allocate
	if len(h.nextExecs) > 0 {
		execs := h.nextExecs[0]
		h.nextExecs = h.nextExecs[1:]
		return execs, nil
	} else {
		return []batonapi.ExecCoordConn{testexec.NewTestExec(h.TestDb), testexec.NewTestExec(h.TestDb)}, nil
	}
}

func (h *TestRig) RepairMissingTxs(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) ([]proto.SignedTransaction, error) {
	panic("not used")
}

func (h *TestRig) ScheduleTxs(ctx context.Context, txs []*proto.SignedTransaction) ([]*proto.SignedTransaction, []*proto.RejectedTransaction) {
	return txs, nil
}

func (h *TestRig) RejectedTx(context.Context, *proto.RejectedTransaction) {}

func (h *TestRig) Committed(context.Context, *proto.CatchupBlock) {}
