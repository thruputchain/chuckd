package sched

import (
	"context"
	"sync"

	proto "github.com/groundhogchain/protocol-go"
)

// type SchedulerConnPool struct {
// 	lk    sync.Mutex
// 	conns map[SchedulerCoordConn]int // conn -> number of outstanding requests
// }

// func ConnectPoolToScheduler() {
// 	XXX
// }

type SchedulerPoolConn struct {
	conn    SchedulerCoordConn
	lk      sync.Mutex
	pending int
}

func (p *SchedulerPoolConn) inc() {
	p.lk.Lock()
	defer p.lk.Unlock()
	p.pending++
}

func (p *SchedulerPoolConn) dec() {
	p.lk.Lock()
	defer p.lk.Unlock()
	p.pending--
}

func (p *SchedulerPoolConn) Schedulercoord_null(ctx context.Context) error {
	XXX
}

func (p *SchedulerPoolConn) Schedulercoord_intro_coord(ctx context.Context, card proto.CoordCard) (proto.CoordinatorRegistrationOutcome, error) {
	XXX
}

func (p *SchedulerPoolConn) Schedulercoord_submit_unvetted(ctx context.Context, txb proto.SignedTransactionBatch) error {
	XXX
}

func (p *SchedulerPoolConn) Schedulercoord_submit_vetted(ctx context.Context, vid proto.ValidatorID, txb proto.SignedTransactionBatch) error {
	XXX
}

func (p *SchedulerPoolConn) Schedulercoord_nominate(ctx context.Context, tip proto.DbTip) (proto.SchedulerNominationResult, error) {
	XXX
}

func (p *SchedulerPoolConn) Schedulercoord_validate(ctx context.Context, tip proto.DbTip, nom proto.Nomination) (proto.SchedulerValidationResult, error) {
	XXX
}

func (p *SchedulerPoolConn) Schedulercoord_commit_block(ctx context.Context, id proto.BlockID) (proto.SchedulerCommitOutcome, error) {
	XXX
}

func (p *SchedulerPoolConn) Schedcoord_get_tip(ctx context.Context) (proto.DbTip, error) {
	XXX
}

func (p *SchedulerPoolConn) Schedcoord_load_genesis_configuration(ctx context.Context, gc proto.GenesisConfiguration) (proto.DbSchedResult, error) {
	XXX
}

func (p *SchedulerPoolConn) Schedcoord_lookup_transactions(ctx context.Context, txh proto.SignedTransactionHashBatch) (proto.OptSignedTransactionBatch, error) {
	XXX
}
