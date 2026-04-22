package baton

import (
	"context"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/catchup"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/test/testcatchup"
	"github.com/groundhogchain/chuckd/test/testcoord"
	"github.com/groundhogchain/chuckd/test/testsched"
	"github.com/groundhogchain/chuckd/test/testutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
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
	*storage.TestCVPersistence
	*cvpool.ConsensusValuePool
	*storage.TestCatchupBlockPersistence
	*catchup.CatchupBlockPool
	//
	rlk  sync.Mutex
	rand *rand.Rand
	//
	nominateRequest  chan nominateIn
	nominateResponse chan bool
	backfeed         chan struct{}
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
		backfeed:         make(chan struct{}),
	}
	h.TestCVPersistence = storage.NewTestCVPersistence()
	h.ConsensusValuePool = cvpool.NewCVPool(ctx, h.TestCVPersistence)
	h.TestCoord = testcoord.NewTestCoord()
	dbDir, err := os.MkdirTemp("", "chuckd-baton-db-*")
	must.NoError(ctx, err)
	h.TestScheduler = testsched.NewTestScheduler(
		ctx,
		signer.Public_info,
		proto.DbTip{Height: 0},
		dbDir,
	)
	// place a consensus value for the db tip
	cv0 := assumedConsensusValueAtHeight(0)
	h.TestCVPersistence.PersistCV(ctx, cv0, protoutil.HashCVFast(&cv0))
	h.TestScheduler.SetCoordScheduler(h.TestCoord)
	catchupSigner, err := protoutil.GenerateIdentity(rand, "test")
	must.NoError(ctx, err)
	//
	h.TestCatchupBlockPersistence = storage.NewTestCatchupBlockPersistence()
	h.CatchupBlockPool = catchup.NewCatchupBlockPool(ctx, h.TestCatchupBlockPersistence)
	h.TestCatchupStaged = testcatchup.NewTestCatchup(time, *catchupSigner)
	return h
}

func (h *TestRig) GetTip(ctx context.Context) proto.DbTip {
	r, err := h.TestScheduler.Schedcoord_get_tip(ctx)
	must.NoError(ctx, err)
	must.Assert(ctx, r.Outcome == proto.SchedulerGetTipOK)
	return *r.Db_tip()
}

func (h *TestRig) GetTipCV(ctx context.Context) (proto.DbTip, proto.ConsensusValue) {
	dbTip := h.GetTip(ctx)
	cvTip := h.FindCommittedCV(ctx, dbTip.Height) // cvTip must be non-nil
	return dbTip, *cvTip
}

func (h *TestRig) RestoreDb(ctx context.Context) (proto.SchedulerGetTipResult, error) {
	return h.Schedcoord_get_tip(ctx)
}

func (h *TestRig) GenNomination(ctx context.Context, ntxs int) *testutil.TestBlock {
	dbTip, cvTip := h.GetTipCV(ctx)
	return h.GenNominationOntoTip(ctx, ntxs, dbTip, cvTip)
}

func (h *TestRig) Bind(b *Baton) {
	h.TestSCP = NewTestSCP(b.SCPHandler())
}

func (h *TestRig) Emit(ctx context.Context, event any) {
	h.TestBus.SendEvent(ctx, h.TestTime.Now(), event)
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

func (h *TestRig) SignXdr(m libxdr.XdrMarshaler, v xdr.XdrType) proto.Signature {
	return protoutil.SignXdr(m, h.signer.Secret_key, v)
}

func (h *TestRig) VerifyXdr(m libxdr.XdrMarshaler, value xdr.XdrType, sig proto.Signature) bool {
	panic("not used")
	// return protoutil.VerifyXdr(m, h.signer.Public_info.Public_key, value, sig)
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

func (h *TestRig) RepairMissingCV(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.ConsensusValueHash,
	height proto.BlockHeight,
) (*proto.ConsensusValue, error) {
	panic("not used")
}

func (h *TestRig) AnnounceNomination(height proto.BlockHeight, cvh proto.ConsensusValueHash) {
}

func (h *TestRig) Committed(ctx context.Context, cb *batonkit.CommittableBlock) {}

// unit tests focus on a single node, so dissemination is not necessary
func (h *TestRig) DisseminateCV(ctx context.Context, cv proto.ConsensusValue, cvh proto.ConsensusValueHash) {
}

func (h *TestRig) PlaceConsensusValue(ctx context.Context, cv proto.ConsensusValue) {
	h.ConsensusValuePool.CVPoolAdd(ctx, cv, protoutil.HashCVFast(&cv))
}

func (h *TestRig) InitSCPLowestNonExternalized(ctx context.Context, slot scpkit.Slot) {
}

func (h *TestRig) BackfeedSCPEnvelopes(ctx context.Context, since scpkit.Slot, timeout time.Duration) {
	h.backfeed <- struct{}{}
}

func (h *TestRig) WaitBackfeed(ctx context.Context) {
	<-h.backfeed
	liblog.Debugf(ctx, "CAPTURED: backfeed request")
}
