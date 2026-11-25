package sched

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func ConnectPoolToScheduler(
	ctx context.Context,
	net libnet.Dialer,
	schedAddr string,
) (SchedulerCoordConn, error) {
	return connectPoolToScheduler(ctx, net, schedAddr, 20)
}

func connectPoolToScheduler(ctx context.Context, net libnet.Dialer, schedAddr string, count int) (SchedulerCoordConn, error) {
	if count < 1 {
		return nil, errors.New("sched: empty connection pool")
	}
	conns := make([]SchedulerCoordConn, count)
	poolCtx, stopPool := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(len(conns))
	for i := range conns {
		go func(i int) {
			defer wg.Done()
			// dialUntilSuccess only returns an error when startup is canceled.
			conns[i], _ = dialUntilSuccess(ctx, schedAddr, func(ctx context.Context) (SchedulerCoordConn, error) {
				return dialSchedulerClient(ctx, poolCtx, net, schedAddr)
			})
		}(i)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		stopPool()
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
		return nil, err
	}
	return newConnPool(poolCtx, stopPool, conns), nil
}

type schedulerDialer func(context.Context) (SchedulerCoordConn, error)

func dialUntilSuccess(ctx context.Context, schedAddr string, dial schedulerDialer) (SchedulerCoordConn, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		conn, err := dial(ctx)
		if err == nil {
			return conn, nil
		}
		liblog.Infof(ctx, "connecting to scheduler at %s failed (%v), retrying in 1 sec", schedAddr, err)
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

type ConnPool struct {
	ctx    context.Context
	cancel context.CancelFunc
	req    chan<- *poolRequest
}

func NewConnPool(ctx context.Context, conns []SchedulerCoordConn) *ConnPool {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	return newConnPool(ctx, cancel, conns)
}

func newConnPool(ctx context.Context, cancel context.CancelFunc, conns []SchedulerCoordConn) *ConnPool {
	rr := make(chan *poolRequest, len(conns))
	p := &ConnPool{ctx: ctx, cancel: cancel, req: rr}
	if len(conns) == 0 {
		cancel()
	}
	for _, conn := range conns {
		go driveConn(ctx, conn, rr)
	}
	return p
}

func (p *ConnPool) Close() { p.cancel() }

// The caller and worker race to claim a queued request. A canceled request
// that has not started can be abandoned safely. Once started, the caller waits
// for the worker to finish using its arguments, even on cancellation.
type poolRequest struct {
	context.Context
	request SchedCoordRequest
	claimed atomic.Bool
}

func poolCall[T any](p *ConnPool, ctx context.Context, queue chan<- *poolRequest, request SchedCoordRequest, reply <-chan T) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if err := p.ctx.Err(); err != nil {
		return zero, err
	}
	callCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(p.ctx, cancel)
	defer stop()
	queued := &poolRequest{Context: callCtx, request: request}
	select {
	case queue <- queued:
	case <-callCtx.Done():
		return zero, callCtx.Err()
	}
	select {
	case r := <-reply:
		return r, nil
	case <-callCtx.Done():
		if queued.claimed.CompareAndSwap(false, true) {
			return zero, callCtx.Err()
		}
		return <-reply, nil
	}
}

func (p *ConnPool) Schedulercoord_null(ctx context.Context) error {
	reply := make(chan Reply_Schedulercoord_null, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedulercoord_null{Reply: reply, Ctx: ctx}, reply)
	if err != nil {
		return err
	}
	return r.Err
}

func (p *ConnPool) Schedulercoord_intro_coord(ctx context.Context, card proto.CoordCard) (proto.CoordinatorRegistrationOutcome, error) {
	reply := make(chan Reply_Schedulercoord_intro_coord, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedulercoord_intro_coord{Reply: reply, Ctx: ctx, CoordCard: card}, reply)
	if err != nil {
		return r.Outcome, err
	}
	return r.Outcome, r.Err
}

func (p *ConnPool) Schedulercoord_submit_unvetted(ctx context.Context, stb proto.ParallelSignedTransactionBatch) error {
	reply := make(chan Reply_Schedulercoord_submit_unvetted, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedulercoord_submit_unvetted{Reply: reply, Ctx: ctx, STB: stb}, reply)
	if err != nil {
		return err
	}
	return r.Err
}

func (p *ConnPool) Schedulercoord_submit_vetted(ctx context.Context, vid proto.ValidatorID, stb proto.ParallelSignedTransactionBatch) error {
	reply := make(chan Reply_Schedulercoord_submit_vetted, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedulercoord_submit_vetted{Reply: reply, Ctx: ctx, VID: vid, STB: stb}, reply)
	if err != nil {
		return err
	}
	return r.Err
}

func (p *ConnPool) Schedulercoord_propose_block(ctx context.Context, onto proto.BlockID, propCx proto.ProposalContext) (proto.SchedulerProposalResult, error) {
	reply := make(chan Reply_Schedulercoord_propose_block, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedulercoord_propose_block{Reply: reply, Ctx: ctx, Onto: onto, PropCx: propCx}, reply)
	if err != nil {
		return r.Result, err
	}
	return r.Result, r.Err
}

func (p *ConnPool) Schedulercoord_validate_block(ctx context.Context, onto proto.BlockID, prop proto.Proposal) (proto.SchedulerValidationResult, error) {
	reply := make(chan Reply_Schedulercoord_validate_block, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedulercoord_validate_block{Reply: reply, Ctx: ctx, Onto: onto, Prop: prop}, reply)
	if err != nil {
		return r.Result, err
	}
	return r.Result, r.Err
}

func (p *ConnPool) Schedulercoord_commit_block(ctx context.Context, id proto.BlockID, value proto.ConsensusValue) (proto.SchedulerCommitOutcome, error) {
	reply := make(chan Reply_Schedulercoord_commit_block, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedulercoord_commit_block{Reply: reply, Ctx: ctx, ID: id, Value: value}, reply)
	if err != nil {
		return r.Outcome, err
	}
	return r.Outcome, r.Err
}

func (p *ConnPool) Schedcoord_get_tip(ctx context.Context) (proto.SchedulerGetTipResult, error) {
	reply := make(chan Reply_Schedcoord_get_tip, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedcoord_get_tip{Reply: reply, Ctx: ctx}, reply)
	if err != nil {
		return r.Tip, err
	}
	return r.Tip, r.Err
}

func (p *ConnPool) Schedcoord_load_genesis_configuration(ctx context.Context, gc proto.GenesisConfiguration) (proto.DbSchedResult, error) {
	reply := make(chan Reply_Schedcoord_load_genesis_configuration, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedcoord_load_genesis_configuration{Reply: reply, Ctx: ctx, GC: gc}, reply)
	if err != nil {
		return r.Result, err
	}
	return r.Result, r.Err
}

func (p *ConnPool) Schedulercoord_commit_genesis_ceremony(ctx context.Context, request proto.GenesisCeremonyRequest) (proto.GenesisCeremonyResult, error) {
	reply := make(chan Reply_Schedulercoord_commit_genesis_ceremony, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedulercoord_commit_genesis_ceremony{Reply: reply, Ctx: ctx, Request: request}, reply)
	if err != nil {
		return r.Result, err
	}
	return r.Result, r.Err
}

func (p *ConnPool) Schedcoord_lookup_transactions(ctx context.Context, sthb proto.SignedTransactionHashBatch) (proto.OptSignedTransactionBatch, error) {
	reply := make(chan Reply_Schedcoord_lookup_transactions, 1)
	r, err := poolCall(p, ctx, p.req, Request_Schedcoord_lookup_transactions{Reply: reply, Ctx: ctx, STHB: sthb}, reply)
	if err != nil {
		return r.OptSTB, err
	}
	return r.OptSTB, r.Err
}

// Schedcoord_lookup_transaction_outcomes exists only to satisfy
// SchedulerCoordConn; chuckd never looks up transaction outcomes.
func (p *ConnPool) Schedcoord_lookup_transaction_outcomes(ctx context.Context, sthb proto.TransactionHashBatch) (proto.SchedulerTransactionOutcomeBatch, error) {
	return nil, errors.New("sched: transaction outcome lookup is not supported by the coordinator pool")
}
