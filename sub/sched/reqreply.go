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
	STB   proto.ParallelSignedTransactionBatch
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

type Request_Schedulercoord_propose_block struct {
	Reply  chan<- Reply_Schedulercoord_propose_block
	Ctx    context.Context
	Onto   proto.BlockID
	PropCx proto.ProposalContext
}

func (Request_Schedulercoord_propose_block) IsSchedCoordRequest() {}

type Reply_Schedulercoord_propose_block struct {
	Result proto.SchedulerProposalResult
	Err    error
}

type Request_Schedulercoord_validate_block struct {
	Reply chan<- Reply_Schedulercoord_validate_block
	Ctx   context.Context
	Onto  proto.BlockID
	Prop  proto.Proposal
}

func (Request_Schedulercoord_validate_block) IsSchedCoordRequest() {}

type Reply_Schedulercoord_validate_block struct {
	Result proto.SchedulerValidationResult
	Err    error
}

type Request_Schedulercoord_commit_block struct {
	Reply chan<- Reply_Schedulercoord_commit_block
	Ctx   context.Context
	ID    proto.BlockID
	Value proto.ConsensusValue
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
	Tip proto.SchedulerGetTipResult
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

type Request_Schedulercoord_commit_genesis_ceremony struct {
	Reply   chan<- Reply_Schedulercoord_commit_genesis_ceremony
	Ctx     context.Context
	Request proto.GenesisCeremonyRequest
}

func (Request_Schedulercoord_commit_genesis_ceremony) IsSchedCoordRequest() {}

type Reply_Schedulercoord_commit_genesis_ceremony struct {
	Result proto.GenesisCeremonyResult
	Err    error
}

type Request_Schedcoord_lookup_transactions struct {
	Reply chan<- Reply_Schedcoord_lookup_transactions
	Ctx   context.Context
	STHB  proto.TransactionHashBatch
}

func (Request_Schedcoord_lookup_transactions) IsSchedCoordRequest() {}

type Reply_Schedcoord_lookup_transactions struct {
	OptSTB proto.OptSignedTransactionBatch
	Err    error
}
