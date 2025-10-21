package baton

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/catchup"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/scp/scpkit"
	"github.com/groundhogchain/chuckd/test/testcatchup"
	"github.com/groundhogchain/chuckd/test/testcoord"
	"github.com/groundhogchain/chuckd/test/testsched"
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
	*testcoord.TestCoord
	*testsched.TestScheduler
	*testcatchup.TestCatchupStaged
	*TestSCP
	//
	rlk  sync.Mutex
	rand *rand.Rand
	//
	nominateRequest  chan nominateIn
	nominateResponse chan bool
	//
	cvlk            sync.Mutex
	consensusValues []proto.ConsensusValue
}

func NewTestRig(
	ctx context.Context,
	rand *rand.Rand,
	now time.Time,
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
		nominateRequest:  make(chan nominateIn, 1),
		nominateResponse: make(chan bool, 1),
	}
	h.TestCoord = testcoord.NewTestCoord()
	h.TestScheduler = testsched.NewTestScheduler(ctx, signer.Public_info, proto.DbTip{})
	h.TestScheduler.SetCoordScheduler(h.TestCoord)
	catchupSigner, err := protoutil.GenerateIdentity(rand, "test")
	must.NoError(ctx, err)
	h.TestCatchupStaged = testcatchup.NewTestCatchup(time, *catchupSigner)
	return h
}

func (h *TestRig) getTip(ctx context.Context) proto.DbTip {
	r, err := h.TestScheduler.Schedcoord_get_tip(ctx)
	must.NoError(ctx, err)
	return *r.Db_tip()
}

func (h *TestRig) RestoreDb(ctx context.Context) (proto.DbTip, error) {
	return h.getTip(ctx), nil
}

func (h *TestRig) GenNomination(ctx context.Context, ntxs int) *testutil.TestBlock {
	return h.GenNominationOntoTip(ctx, ntxs, h.getTip(ctx))
}

func (h *TestRig) Bind(b *Baton) {
	h.TestSCP = NewTestSCP(b.SCPHandler())
}

func (h *TestRig) Emit(ctx context.Context, event any) {
	h.TestBus.SendEvent(h.TestTime.Now(), event)
}

type nominateIn struct {
	Slot  scpkit.Slot
	Value xdr.XdrType
	Prev  xdr.XdrType
}

func (h *TestRig) Nominate(ctx context.Context, slot scpkit.Slot, value xdr.XdrType, prev xdr.XdrType) bool {
	h.nominateRequest <- nominateIn{Slot: slot, Value: value, Prev: prev}
	return <-h.nominateResponse
}

func (h *TestRig) WaitNomination(ctx context.Context) (slot scpkit.Slot, value xdr.XdrType, prev xdr.XdrType) {
	x := <-h.nominateRequest
	return x.Slot, x.Value, x.Prev
}

func (h *TestRig) SignalNomination(result bool) {
	h.nominateResponse <- result
}

func (h *TestRig) StopNomination(ctx context.Context, slot scpkit.Slot) {
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

func (h *TestRig) GenerateID64(context.Context) batonkit.ID64 {
	h.rlk.Lock()
	defer h.rlk.Unlock()
	return batonkit.ID64(h.rand.Int63())
}

func (h *TestRig) DialCatchup(context.Context, proto.ValidatorPublicInfo) (catchup.CatchupConn, error) {
	return h.TestCatchupStaged, nil
}

func (h *TestRig) RepairMissingTxs(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) ([]proto.SignedTransaction, error) {
	panic("not used")
}

func (h *TestRig) PersistConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	h.cvlk.Lock()
	defer h.cvlk.Unlock()
	h.consensusValues = append(h.consensusValues, cv)
}

func (h *TestRig) FindLatestConsensusValues(ctx context.Context, n int) ([]proto.ConsensusValue, error) {
	h.cvlk.Lock()
	defer h.cvlk.Unlock()
	m := len(h.consensusValues)
	return h.consensusValues[max(0, m-n):m], nil
}
