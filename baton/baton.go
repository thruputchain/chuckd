package baton

import (
	"context"
	"log"
	"sync"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func NewBaton(ctx context.Context, h Handler) *Baton {
	b := &Baton{
		handler:      h,
		dbRestored:   make(chan dbRestored, 1),
		externalized: make(chan proposal),
		caughtUp:     make(chan caughtUp),
		committed:    make(chan committed),
		commit:       make(chan *ValidatableBlock),
		nominate:     make(chan *nominate),
	}
	b.catchup.pending = map[ID64]catchupRequest{}
	b.catchup.blocks = nil

	go b.startBaton(ctx)
	return b
}

type Baton struct {
	handler Handler
	// notifications to baton
	dbRestored   chan dbRestored
	externalized chan proposal
	caughtUp     chan caughtUp
	committed    chan committed
	// notifications from baton
	commit   chan *ValidatableBlock
	nominate chan *nominate
	// states
	db struct {
		lk        sync.Mutex
		tipHeight proto.BlockHeight
		tipHash   proto.BlockHash
		tipID     proto.BlockID
	}
	catchup struct {
		lk      sync.Mutex
		pending map[ID64]catchupRequest // pending catchup requests
		blocks  catchupBlocks           // received catchup blocks
	}
	consensus struct {
		lk   sync.Mutex
		prop proposals
	}
	nominator struct {
		lk             sync.Mutex
		latestProposal *proto.Proposal
		latestBlockID  proto.BlockID
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

	// start nominator who depends on the db state
	go b.startNominating(ctx)

	// start validate and commit thread
	go b.startValidateAndCommit(ctx)

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

		// validate and commit next block, if one is available and the validate and commit thread is not busy
		if vb := v.NextBlockToValidateAndCommit(); vb != nil {
			select {
			case b.commit <- vb:
			default:
			}
		}

		// send catchup requests
		b.requestCatchup(ctx, v.NeededHeightSpans())

		// wait until something happens
		select {
		case <-b.caughtUp: // catchup blocks received
		case <-b.externalized: // externalization received
		case <-b.committed: // block committed to db
		}
	}
}
