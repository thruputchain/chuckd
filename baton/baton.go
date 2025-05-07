package baton

import (
	"context"
	"log"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/xid"
)

func NewBaton(ctx context.Context, h Handler) *Baton {
	b := &Baton{
		handler:      h,
		dbRestored:   make(chan dbRestored, 1),
		externalized: make(chan proposal),
		caughtUp:     make(chan caughtUp),
	}
	b.catchup.pending = map[xid.ID]catchupRequest{}
	b.catchup.blocks = nil

	go b.startBaton(ctx)
	return b
}

type Baton struct {
	handler Handler
	// callbacks
	dbRestored   chan dbRestored
	externalized chan proposal
	caughtUp     chan caughtUp
	// states
	db struct { // local db
		lk        sync.Mutex
		tipHeight proto.BlockHeight
		tipHash   proto.BlockHash
	}
	catchup struct { // catching up
		lk      sync.Mutex
		pending map[xid.ID]catchupRequest // pending catchup requests
		blocks  catchupBlocks             // received catchup blocks
	}
	scp struct { // scp view
		lk   sync.Mutex
		prop proposals
	}
	prop struct { // proposing
		lk sync.Mutex
	}
}

func (b *Baton) startBaton(ctx context.Context) {

	// externalized chan is active right away

	// restore db
	go b.restoreDb(ctx)
	select {
	case r := <-b.dbRestored:
		if r.Err != nil {
			log.Fatalf("db not restored: %v", r.Err)
		}
	case <-b.externalized:
		// ignore (externalizations are already recorded in the state)
	}

	// start proposer who depends only on the db state (and txs in mempool)
	go b.startNominating(ctx)

	for {

		// build view
		v := (*view)(nil)
		for v == nil {
			var err error
			v, err = b.buildView(ctx)
			switch err {
			case nil:
				must.Assertf(ctx, v != nil, "")
			case ErrConsensusBelowDb: // unrecoverable
				log.Fatal("consensus height is below db; unrecoverable")
			case ErrDbFarBehind: // must resort to cold start
				log.Fatalf("db is far behind consensus; needs a cold start")
			case ErrDbConsensusDisagree: // unrecoverable
				log.Fatal("consensus and db disagree; unrecoverable")
			case ErrConsensusInconsistent: // unrecoverable
				log.Fatal("consensus is inconsistent; unrecoverable")
			case ErrFaultyCatchup: // must drop all catchup blocks and blacklist sources
				b.purgeCatchup(ctx) // go back and retry building the view
			default:
				panic("unexpected")
			}
		}

		// send catchup requests
		b.requestCatchup(ctx, v.NeededHeightSpans())

		// commit to db if possible
		// panic("XXX")

		select {
		case <-b.caughtUp: // catchup blocks received
		case <-b.externalized: // externalization received
		}

		panic("XXX")
	}
}
