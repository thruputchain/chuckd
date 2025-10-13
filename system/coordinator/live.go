package coordinator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/catchup"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/repair"
	"github.com/groundhogchain/chuckd/sched"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/jay/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

// LiveHandler is an batonkit.Handler that connects a live system to the baton
type LiveHandler struct {
	protoutil.Signatory
	libtime.Time
	inside  libnet.Network
	outside libnet.Network
	sched   sched.SchedulerConn
	router  router.Router
	*repair.RepairAgent
	dogwood *dogwood.DogwoodService
	scp     scpapi.SCPController
	m       xdrutil.XdrMarshaler
	//
	rlk  sync.Mutex
	rand *rand.Rand
	//
	scpReady   chan struct{}
	schedReady chan struct{}
}

func NewLiveHandler(
	signer protoutil.Signatory,
	time libtime.Time,
	inside libnet.Network,
	outside libnet.Network,
	router router.Router,
	repairAgent *repair.RepairAgent,
	dogwood *dogwood.DogwoodService,
	rand *rand.Rand,
) *LiveHandler {
	return &LiveHandler{
		Signatory:   signer,
		Time:        time,
		inside:      inside,
		outside:     outside,
		router:      router,
		RepairAgent: repairAgent,
		dogwood:     dogwood,
		m:           xdrutil.NewXdrMarshaler(),
		rand:        rand,
		scpReady:    make(chan struct{}),
		schedReady:  make(chan struct{}),
	}
}

func (x *LiveHandler) SetSCP(scp scpapi.SCPController) {
	x.scp = scp
	close(x.scpReady)
}

func (x *LiveHandler) SetScheduler(conn sched.SchedulerConn) {
	x.sched = conn
	close(x.schedReady)
}

func (x *LiveHandler) RestoreDb(ctx context.Context) (proto.DbTip, error) {
	<-x.schedReady

	tipResult, err := x.sched.Schedcoord_get_tip(ctx)
	must.NoError(ctx, err)
	if tipResult.Result != proto.DbSchedResultOK {
		return batonkit.Tip{}, fmt.Errorf("sched get tip error")
	}
	tip := *tipResult.Db_tip()
	return tip, nil
}

func (x *LiveHandler) Nominate(ctx context.Context, slot scpapi.Slot, value xdr.XdrType, prev xdr.XdrType) bool {
	<-x.scpReady
	return x.scp.Nominate(ctx, slot, x.m.MarshalXdr(value).Bytes(), x.m.MarshalXdr(prev).Bytes())
}

func (x *LiveHandler) StopNomination(ctx context.Context, slot scpapi.Slot) {
	<-x.scpReady
	x.scp.StopNomination(ctx, slot)
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

func (x *LiveHandler) Emit(ctx context.Context, event any) {
	liblog.Debugf("%s", event)
}
