package batonkit

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/sub/catchup"
	"github.com/groundhogchain/chuckd/sub/cvpool"
	"github.com/groundhogchain/chuckd/sub/repair/repairkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpbroker"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/util/protoutil"
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
	storage.ConsensusValuePersistence
	cvpool.CVPool
	scpbroker.SCPBackfeed
	catchup.CatchupAgent

	DisseminateCV(context.Context, proto.ConsensusValue)

	GenerateID64(context.Context) ID64
	RestoreDb(context.Context) (proto.DbTip, error)

	Emit(ctx context.Context, event any)
}

type ID64 int64

func (id ID64) String() string { return fmt.Sprintf("%x", int64(id)) }

type ConsensusController interface {
	Nominate(ctx context.Context, slot scpkit.Slot, value xdr.XdrType, prev xdr.XdrType) bool
	StopNomination(ctx context.Context, slot scpkit.Slot)
}

type BlockHandler interface {
	Committed(ctx context.Context, height proto.BlockHeight)
}
