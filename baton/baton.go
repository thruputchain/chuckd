package baton

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/log"
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
	b.consensus.validatedProposal = map[proto.BlockHash]*proto.Proposal{}
	b.consensus.validatedBlockID = map[proto.BlockHash]*proto.BlockID{}
	return b
}

func (b *Baton) Start(ctx context.Context) {
	go b.startBaton(ctx)
}

type Baton struct {
	handler Handler
	// notifications to baton
	dbRestored   chan dbRestored
	externalized chan proposal // from scp handler
	caughtUp     chan caughtUp
	committed    chan committed
	// signals from baton
	commit   chan *ValidatableBlock // to commit process
	nominate chan *nominate         // to nomination process
	// states
	db struct {
		lk  sync.Mutex
		tip Tip
	}
	catchup struct {
		lk      sync.Mutex
		pending map[ID64]catchupRequest // pending catchup requests
		blocks  catchupBlocks           // received catchup blocks
	}
	consensus struct {
		lk                sync.Mutex
		ext               proposals
		validatedProposal map[proto.BlockHash]*proto.Proposal
		validatedBlockID  map[proto.BlockHash]*proto.BlockID
	}
	nominator struct {
		lk                sync.Mutex
		nominatedProposal *proto.Proposal
		nominatedBlockID  *proto.BlockID
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
		log.Debugf("<-b.dbRestored")
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
				log.Fatalf("consensus height is below db; unrecoverable")
			case ErrDbFarBehind: // must resort to cold start
				log.Fatalf("db is far behind consensus; needs a cold start")
			case ErrDbConsensusDisagree: // unrecoverable
				log.Fatalf("consensus and db disagree; unrecoverable")
			case ErrConsensusInconsistent: // unrecoverable
				log.Fatalf("consensus is inconsistent; unrecoverable")
			case ErrFaultyCatchup: // must drop all catchup blocks and blacklist sources
				log.Errorf("purging faulty catchup")
				b.purgeCatchup(ctx) // go back and retry building the view
			default:
				panic("unexpected")
			}
		}

		// validate and commit next block, if one is available and the validate and commit thread is not busy
		if vb := v.NextBlockToCommit(); vb != nil {
			select {
			case b.commit <- vb: // signal may be ignored, if we are currently committing
			default:
			}
		}

		// start nomination, if the db there are no externalizations above the db tip
		if nom := v.ReadyToNominate(); nom != nil {
			b.nominate <- nom // signal interrupts any ongoing nomination
		}

		// send catchup requests
		if needed := v.NeededUnrequestedSpans(); needed.IsNotEmpty() {
			b.requestCatchup(ctx, needed)
		}

		// wait until something happens
		select {
		case <-b.caughtUp: // catchup blocks received
			log.Debugf("<-b.caughtUp")
		case <-b.externalized: // externalization received
			log.Debugf("<-b.externalized")
		case <-b.committed: // block committed to db
			log.Debugf("<-b.committed")
		}
	}
}
