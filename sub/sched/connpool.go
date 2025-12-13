package sched

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

// type SchedulerConnPool struct {
// 	lk    sync.Mutex
// 	conns map[SchedulerCoordConn]int // conn -> number of outstanding requests
// }

// func ConnectPoolToScheduler() {
// 	XXX
// }

type ConnPool struct {
	req chan<- SchedCoordRequest
}

func (p *ConnPool) Schedulercoord_null(ctx context.Context) error {
	reply := make(chan Reply_Schedulercoord_null, 1)
	p.req <- Request_Schedulercoord_null{Reply: reply, Ctx: ctx}
	return (<-reply).Err
}

func (p *ConnPool) Schedulercoord_intro_coord(ctx context.Context, card proto.CoordCard) (proto.CoordinatorRegistrationOutcome, error) {
	reply := make(chan Reply_Schedulercoord_intro_coord, 1)
	p.req <- Request_Schedulercoord_intro_coord{Reply: reply, Ctx: ctx, CoordCard: card}
	r := <-reply
	return r.Outcome, r.Err
}

func (p *ConnPool) Schedulercoord_submit_unvetted(ctx context.Context, stb proto.SignedTransactionBatch) error {
	reply := make(chan Reply_Schedulercoord_submit_unvetted, 1)
	p.req <- Request_Schedulercoord_submit_unvetted{Reply: reply, Ctx: ctx, STB: stb}
	r := <-reply
	return r.Err
}

func (p *ConnPool) Schedulercoord_submit_vetted(ctx context.Context, vid proto.ValidatorID, stb proto.SignedTransactionBatch) error {
	reply := make(chan Reply_Schedulercoord_submit_vetted, 1)
	p.req <- Request_Schedulercoord_submit_vetted{Reply: reply, Ctx: ctx, VID: vid, STB: stb}
	r := <-reply
	return r.Err
}

func (p *ConnPool) Schedulercoord_nominate(ctx context.Context, tip proto.DbTip) (proto.SchedulerNominationResult, error) {
	reply := make(chan Reply_Schedulercoord_nominate, 1)
	p.req <- Request_Schedulercoord_nominate{Reply: reply, Ctx: ctx, Tip: tip}
	r := <-reply
	return r.Result, r.Err
}

func (p *ConnPool) Schedulercoord_validate(ctx context.Context, tip proto.DbTip, nom proto.Nomination) (proto.SchedulerValidationResult, error) {
	reply := make(chan Reply_Schedulercoord_validate, 1)
	p.req <- Request_Schedulercoord_validate{Reply: reply, Ctx: ctx, Tip: tip, Nom: nom}
	r := <-reply
	return r.Result, r.Err
}

func (p *ConnPool) Schedulercoord_commit_block(ctx context.Context, id proto.BlockID) (proto.SchedulerCommitOutcome, error) {
	reply := make(chan Reply_Schedulercoord_commit_block, 1)
	p.req <- Request_Schedulercoord_commit_block{Reply: reply, Ctx: ctx, ID: id}
	r := <-reply
	return r.Outcome, r.Err
}

func (p *ConnPool) Schedcoord_get_tip(ctx context.Context) (proto.DbTip, error) {
	reply := make(chan Reply_Schedcoord_get_tip, 1)
	p.req <- Request_Schedcoord_get_tip{Reply: reply, Ctx: ctx}
	r := <-reply
	return r.Tip, r.Err
}

func (p *ConnPool) Schedcoord_load_genesis_configuration(ctx context.Context, gc proto.GenesisConfiguration) (proto.DbSchedResult, error) {
	reply := make(chan Reply_Schedcoord_load_genesis_configuration, 1)
	p.req <- Request_Schedcoord_load_genesis_configuration{Reply: reply, GC: gc}
	r := <-reply
	return r.Result, r.Err
}

func (p *ConnPool) Schedcoord_lookup_transactions(ctx context.Context, sthb proto.SignedTransactionHashBatch) (proto.OptSignedTransactionBatch, error) {
	reply := make(chan Reply_Schedcoord_lookup_transactions, 1)
	p.req <- Request_Schedcoord_lookup_transactions{Reply: reply, STHB: sthb}
	r := <-reply
	return r.OptSTB, r.Err
}
