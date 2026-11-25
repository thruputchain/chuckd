package sched

import (
	"context"
)

func driveConn(
	ctx context.Context,
	conn SchedulerCoordConn,
	req <-chan *poolRequest,
) error {
	defer conn.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case queued := <-req:
			if err := ctx.Err(); err != nil {
				return err
			}
			if queued.Context.Err() != nil || !queued.claimed.CompareAndSwap(false, true) {
				continue
			}
			switch req := queued.request.(type) {

			case Request_Schedulercoord_null:
				err := conn.Schedulercoord_null(queued.Context)
				req.Reply <- Reply_Schedulercoord_null{Err: err}

			case Request_Schedulercoord_intro_coord:
				outcome, err := conn.Schedulercoord_intro_coord(queued.Context, req.CoordCard)
				req.Reply <- Reply_Schedulercoord_intro_coord{Outcome: outcome, Err: err}

			case Request_Schedulercoord_submit_unvetted:
				err := conn.Schedulercoord_submit_unvetted(queued.Context, req.STB)
				req.Reply <- Reply_Schedulercoord_submit_unvetted{Err: err}

			case Request_Schedulercoord_submit_vetted:
				err := conn.Schedulercoord_submit_vetted(queued.Context, req.VID, req.STB)
				req.Reply <- Reply_Schedulercoord_submit_vetted{Err: err}

			case Request_Schedulercoord_propose_block:
				result, err := conn.Schedulercoord_propose_block(queued.Context, req.Onto, req.PropCx)
				req.Reply <- Reply_Schedulercoord_propose_block{Result: result, Err: err}

			case Request_Schedulercoord_validate_block:
				result, err := conn.Schedulercoord_validate_block(queued.Context, req.Onto, req.Prop)
				req.Reply <- Reply_Schedulercoord_validate_block{Result: result, Err: err}

			case Request_Schedulercoord_commit_block:
				outcome, err := conn.Schedulercoord_commit_block(queued.Context, req.ID, req.Value)
				req.Reply <- Reply_Schedulercoord_commit_block{Outcome: outcome, Err: err}

			case Request_Schedcoord_get_tip:
				tipResult, err := conn.Schedcoord_get_tip(queued.Context)
				req.Reply <- Reply_Schedcoord_get_tip{Tip: tipResult, Err: err}

			case Request_Schedcoord_load_genesis_configuration:
				result, err := conn.Schedcoord_load_genesis_configuration(queued.Context, req.GC)
				req.Reply <- Reply_Schedcoord_load_genesis_configuration{Result: result, Err: err}

			case Request_Schedulercoord_commit_genesis_ceremony:
				result, err := conn.Schedulercoord_commit_genesis_ceremony(queued.Context, req.Request)
				req.Reply <- Reply_Schedulercoord_commit_genesis_ceremony{Result: result, Err: err}

			case Request_Schedcoord_lookup_transactions:
				optSTB, err := conn.Schedcoord_lookup_transactions(queued.Context, req.STHB)
				req.Reply <- Reply_Schedcoord_lookup_transactions{OptSTB: optSTB, Err: err}

			default:
				panic("unknown sched_coord request type")
			}
		}
	}
}
