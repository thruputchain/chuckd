package sched

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type SchedCoordRequest interface {
	IsSchedCoordRequest()
}

type Request_Schedulercoord_null struct {
	Reply chan<- Reply_Schedulercoord_null
	Ctx   context.Context
}

func (Request_Schedulercoord_null) IsSchedCoordRequest() {}

type Reply_Schedulercoord_null struct {
	Err error
}

type Request_Schedulercoord_intro_coord struct {
	Reply     chan<- Reply_Schedulercoord_intro_coord
	Ctx       context.Context
	CoordCard proto.CoordCard
}

func (Request_Schedulercoord_intro_coord) IsSchedCoordRequest() {}

type Reply_Schedulercoord_intro_coord struct {
	Outcome proto.CoordinatorRegistrationOutcome
	Err     error
}

type Request_Schedulercoord_submit_unvetted struct {
	Reply chan<- Reply_Schedulercoord_submit_unvetted
	Ctx   context.Context
	STB   proto.SignedTransactionBatch
}

func (Request_Schedulercoord_submit_unvetted) IsSchedCoordRequest() {}

type Reply_Schedulercoord_submit_unvetted struct {
	Err error
}

type Request_Schedulercoord_submit_vetted struct {
	Reply chan<- Reply_Schedulercoord_submit_vetted
	Ctx   context.Context
	VID   proto.ValidatorID
	STB   proto.ParallelSignedTransactionBatch
}

func (Request_Schedulercoord_submit_vetted) IsSchedCoordRequest() {}

type Reply_Schedulercoord_submit_vetted struct {
	Err error
}

type Request_Schedulercoord_nominate struct {
	Reply chan<- Reply_Schedulercoord_nominate
	Ctx   context.Context
	Tip   proto.DbTip
}

func (Request_Schedulercoord_nominate) IsSchedCoordRequest() {}

type Reply_Schedulercoord_nominate struct {
	Result proto.SchedulerNominationResult
	Err    error
}

type Request_Schedulercoord_validate struct {
	Reply chan<- Reply_Schedulercoord_validate
	Ctx   context.Context
	Tip   proto.DbTip
	Nom   proto.Nomination
}

func (Request_Schedulercoord_validate) IsSchedCoordRequest() {}

type Reply_Schedulercoord_validate struct {
	Result proto.SchedulerValidationResult
	Err    error
}

type Request_Schedulercoord_commit_block struct {
	Reply chan<- Reply_Schedulercoord_commit_block
	Ctx   context.Context
	ID    proto.BlockID
}

func (Request_Schedulercoord_commit_block) IsSchedCoordRequest() {}

type Reply_Schedulercoord_commit_block struct {
	Outcome proto.SchedulerCommitOutcome
	Err     error
}

type Request_Schedcoord_get_tip struct {
	Reply chan<- Reply_Schedcoord_get_tip
	Ctx   context.Context
}

func (Request_Schedcoord_get_tip) IsSchedCoordRequest() {}

type Reply_Schedcoord_get_tip struct {
	Tip proto.DbTip
	Err error
}

type Request_Schedcoord_load_genesis_configuration struct {
	Reply chan<- Reply_Schedcoord_load_genesis_configuration
	Ctx   context.Context
	GC    proto.GenesisConfiguration
}

func (Request_Schedcoord_load_genesis_configuration) IsSchedCoordRequest() {}

type Reply_Schedcoord_load_genesis_configuration struct {
	Result proto.DbSchedResult
	Err    error
}

type Request_Schedcoord_lookup_transactions struct {
	Reply chan<- Reply_Schedcoord_lookup_transactions
	Ctx   context.Context
	STHB  proto.SignedTransactionHashBatch
}

func (Request_Schedcoord_lookup_transactions) IsSchedCoordRequest() {}

type Reply_Schedcoord_lookup_transactions struct {
	OptSTB proto.OptSignedTransactionBatch
	Err    error
}
