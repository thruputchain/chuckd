package batonkit

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/catchup"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/sched"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type Handler interface {
	protoutil.Signatory
	libtime.Time
	GenerateID64(context.Context) ID64

	Consensus
	sched.SchedulerConn

	RestoreDb(context.Context) (proto.DbTip, error)

	GetCatchupSources(context.Context, int64) []proto.ValidatorPublicInfo
	DialCatchup(context.Context, proto.ValidatorPublicInfo) (catchup.CatchupConn, error)

	RepairMissingTxs(
		ctx context.Context,
		from proto.ValidatorPublicInfo,
		missing proto.MissingTransactions,
	) ([]proto.SignedTransaction, error)

	Emit(ctx context.Context, event any)
}

type ID64 int64

func (id ID64) String() string { return fmt.Sprintf("%x", int64(id)) }

type Consensus interface {
	Nominate(ctx context.Context, slot scpapi.Slot, value xdr.XdrType, prev xdr.XdrType) bool
	StopNomination(ctx context.Context, slot scpapi.Slot)
}
