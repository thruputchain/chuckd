package sched

import (
	"context"
)

func driveConn(
	ctx context.Context,
	conn SchedulerCoordConn,
	req <-chan SchedCoordRequest,
) error {
	defer conn.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req := <-req:
			switch req := req.(type) {

			case Request_Schedulercoord_null:
				err := conn.Schedulercoord_null(req.Ctx)
				req.Reply <- Reply_Schedulercoord_null{Err: err}

			case Request_Schedulercoord_intro_coord:
				outcome, err := conn.Schedulercoord_intro_coord(req.Ctx, req.CoordCard)
				req.Reply <- Reply_Schedulercoord_intro_coord{Outcome: outcome, Err: err}

			case Request_Schedulercoord_submit_unvetted:
				err := conn.Schedulercoord_submit_unvetted(req.Ctx, req.STB)
				req.Reply <- Reply_Schedulercoord_submit_unvetted{Err: err}

			case Request_Schedulercoord_submit_vetted:
				err := conn.Schedulercoord_submit_vetted(req.Ctx, req.VID, req.STB)
				req.Reply <- Reply_Schedulercoord_submit_vetted{Err: err}

			case Request_Schedulercoord_propose_block:
				result, err := conn.Schedulercoord_propose_block(req.Ctx, req.Onto, req.PropCx)
				req.Reply <- Reply_Schedulercoord_propose_block{Result: result, Err: err}

			case Request_Schedulercoord_validate_block:
				result, err := conn.Schedulercoord_validate_block(req.Ctx, req.Onto, req.Prop)
				req.Reply <- Reply_Schedulercoord_validate_block{Result: result, Err: err}

			case Request_Schedulercoord_commit_block:
				outcome, err := conn.Schedulercoord_commit_block(req.Ctx, req.ID, req.Value)
				req.Reply <- Reply_Schedulercoord_commit_block{Outcome: outcome, Err: err}

			case Request_Schedcoord_get_tip:
				tipResult, err := conn.Schedcoord_get_tip(req.Ctx)
				req.Reply <- Reply_Schedcoord_get_tip{Tip: tipResult, Err: err}

			case Request_Schedcoord_load_genesis_configuration:
				result, err := conn.Schedcoord_load_genesis_configuration(req.Ctx, req.GC)
				req.Reply <- Reply_Schedcoord_load_genesis_configuration{Result: result, Err: err}

			case Request_Schedulercoord_commit_genesis_ceremony:
				result, err := conn.Schedulercoord_commit_genesis_ceremony(req.Ctx, req.Request)
				req.Reply <- Reply_Schedulercoord_commit_genesis_ceremony{Result: result, Err: err}

			case Request_Schedcoord_lookup_transactions:
				optSTB, err := conn.Schedcoord_lookup_transactions(req.Ctx, req.STHB)
				req.Reply <- Reply_Schedcoord_lookup_transactions{OptSTB: optSTB, Err: err}

			default:
				panic("unknown sched_coord request type")
			}
		}
	}
}
