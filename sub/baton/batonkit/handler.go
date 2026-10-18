package batonkit

import (
	"context"
	"fmt"

	"github.com/thruputchain/chuckd/sub/catchup"
	"github.com/thruputchain/chuckd/sub/cvpool"
	"github.com/thruputchain/chuckd/sub/repair/repairkit"
	"github.com/thruputchain/chuckd/sub/repair/repairservice"
	"github.com/thruputchain/chuckd/sub/scp/scpkit"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/libtime"
	proto "github.com/thruputchain/thruput-protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type Handler interface {
	protoutil.Signatory
	libtime.Time
	ConsensusController
	BlockHandler
	proto.SchedulerCoord1
	repairkit.RepairAgent
	repairservice.RepairProvider
	cvpool.CVPool
	catchup.CBPool
	scpkit.SCPCommander
	catchup.CatchupAgent

	DisseminateBasis(context.Context, proto.ConsensusBasis, proto.ConsensusValue)

	GenerateID64(context.Context) ID64
	RestoreDb(context.Context) (proto.SchedulerGetTipResult, error)

	Emit(ctx context.Context, event any)
}

type ConsensusController interface {
	SubmitNomination(ctx context.Context, slot scpkit.Slot, value xdr.XdrType, prev xdr.XdrType) bool
	AbortNomination(ctx context.Context, slot scpkit.Slot)
}

type BlockHandler interface {
	Committed(ctx context.Context, cb *CommittableBlock)
}

type ID64 int64

func (id ID64) String() string { return fmt.Sprintf("%x", int64(id)) }
