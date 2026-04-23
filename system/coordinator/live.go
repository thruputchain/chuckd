package coordinator

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/net/multiplex"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/catchup"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/repair/repairagent"
	"github.com/groundhogchain/chuckd/sub/repair/repairservice"
	"github.com/groundhogchain/chuckd/sub/sched"
	"github.com/groundhogchain/chuckd/sub/scp/scpbroker"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

// LiveHandler is an batonkit.Handler that connects a live system to the baton
type LiveHandler struct {
	notify Notify
	protoutil.Signatory
	libtime.Time
	inside  libnet.Network
	outside libnet.Network
	sched.SchedulerCoordConn
	router router.Router
	*repairagent.RepairAgent
	*repairservice.RepairService
	*cvpool.ConsensusValuePool
	*catchup.CatchupBlockPool
	multiplex     *multiplex.Multiplex
	storage       *storage.DiskStorage
	scpBroker     *scpbroker.SCPBroker
	scpController scpkit.SCPController
	m             libxdr.XdrMarshaler
	//
	rlk  sync.Mutex
	rand *rand.Rand
	//
	scpReady   chan struct{}
	schedReady chan struct{}
}

func NewLiveHandler(
	notify Notify,
	signer protoutil.Signatory,
	time libtime.Time,
	inside libnet.Network,
	outside libnet.Network,
	router router.Router,
	repairAgent *repairagent.RepairAgent,
	repairService *repairservice.RepairService,
	cvpool *cvpool.ConsensusValuePool,
	catchup *catchup.CatchupBlockPool,
	multiplex *multiplex.Multiplex,
	storage *storage.DiskStorage,
	scpBroker *scpbroker.SCPBroker,
	rand *rand.Rand,
) *LiveHandler {
	return &LiveHandler{
		notify:             notify,
		Signatory:          signer,
		Time:               time,
		inside:             inside,
		outside:            outside,
		router:             router,
		RepairAgent:        repairAgent,
		RepairService:      repairService,
		ConsensusValuePool: cvpool,
		CatchupBlockPool:   catchup,
		multiplex:          multiplex,
		storage:            storage,
		scpBroker:          scpBroker,
		m:                  libxdr.NewXdrMarshaler(),
		rand:               rand,
		scpReady:           make(chan struct{}),
		schedReady:         make(chan struct{}),
	}
}

func (x *LiveHandler) SetSCP(scp scpkit.SCPController) {
	x.scpController = scp
	close(x.scpReady)
}

func (x *LiveHandler) SetScheduler(conn sched.SchedulerCoordConn) {
	x.SchedulerCoordConn = conn
	close(x.schedReady)
}

func (x *LiveHandler) RestoreDb(ctx context.Context) (proto.SchedulerGetTipResult, error) {
	<-x.schedReady

	tipResult, err := x.Schedcoord_get_tip(ctx)
	must.NoError(ctx, err)
	return tipResult, nil
}

func (x *LiveHandler) Nominate(ctx context.Context, slot scpkit.Slot, value xdr.XdrType, prev xdr.XdrType) bool {
	<-x.scpReady
	valueBuf := x.m.MarshalXdr(value)
	defer valueBuf.Recycle()
	prevBuf := x.m.MarshalXdr(prev)
	defer prevBuf.Recycle()
	return x.scpController.Nominate(ctx, slot, valueBuf.Bytes(), prevBuf.Bytes())
}

func (x *LiveHandler) StopNomination(ctx context.Context, slot scpkit.Slot) {
	<-x.scpReady
	x.scpController.StopNomination(ctx, slot)
}

func (x *LiveHandler) GetCatchupSources(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	<-x.scpReady
	return x.router.ChooseCatchupProviders(ctx, seed)
}

func (x *LiveHandler) DialCatchup(ctx context.Context, v proto.ValidatorPublicInfo) (catchup.CatchupConn, error) {
	<-x.scpReady
	return catchup.DialCatchupClient(ctx, x.outside, v.Address)
}

func (x *LiveHandler) GenerateID64(context.Context) batonkit.ID64 {
	x.rlk.Lock()
	defer x.rlk.Unlock()
	return batonkit.ID64(x.rand.Int63())
}

func (x *LiveHandler) Emit(ctx context.Context, event any) {}

func (x *LiveHandler) Committed(ctx context.Context, cb *batonkit.CommittableBlock) {
	x.ConsensusValuePool.PurgeNotAbove(ctx, cb.BlockHeight())
	x.notify.Committed(ctx, cb)
}

func (x *LiveHandler) DisseminateCV(ctx context.Context, cv proto.ConsensusValue, cvh proto.ConsensusValueHash) {
	x.ConsensusValuePool.CVPoolAdd(ctx, cv, cvh)
	x.multiplex.SendConsensusValue(ctx, cv)
}

func (x *LiveHandler) BackfeedSCPEnvelopes(ctx context.Context, since scpkit.Slot, timeout time.Duration) {
	x.scpBroker.BackfeedSCPEnvelopes(ctx, since, timeout)
}

func (x *LiveHandler) InitSCPLowestNonExternalized(ctx context.Context, slot scpkit.Slot) {
	x.scpBroker.InitSCPLowestNonExternalized(ctx, slot)
}
