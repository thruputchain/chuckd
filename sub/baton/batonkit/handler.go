package batonkit

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/sub/catchup"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/repair/repairkit"
	"github.com/groundhogchain/chuckd/sub/repair/repairservice"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libtime"
	proto "github.com/groundhogchain/protocol-go"
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

	DisseminateCV(context.Context, proto.ConsensusValue, proto.ConsensusValueCue)

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
