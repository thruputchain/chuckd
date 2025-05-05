package baton

import (
	"context"
	"log"
	"sync"

	proto "github.com/groundhogchain/protocol-go"
)

func NewBaton(ctx context.Context, h Handler) *Baton {
	b := &Baton{
		handler:    h,
		dbRestored: make(chan error, 1),
		extProp:    make(chan externalizedProposal),
		caughtUp:   make(chan ping),
	}
	go b.startBaton(ctx)
	// start mempool
	// ...
	return b
}

type ping struct{}

type Baton struct {
	handler Handler
	// callbacks
	dbRestored chan error
	extProp    chan externalizedProposal
	caughtUp   chan ping
	// states
	lk sync.Mutex
	db struct { // local db
		lk          sync.Mutex
		dbTipHeight proto.BlockHeight
		dbTipHash   proto.BlockHash
	}
	catchup struct { // catching up
		lk     sync.Mutex
		blocks map[proto.ValidatorID]*catchupBlocks
	}
	scp struct { // scp view
		lk   sync.Mutex
		prop proposals
	}
	prop struct { // proposing
		lk sync.Mutex
	}
}

func (b *Baton) restoreDb(ctx context.Context) {
	b.dbRestored <- b.handler.RestoreDb(ctx)
}

func (b *Baton) startBaton(ctx context.Context) {

	// extProp is active right away

	// restore db
	go b.restoreDb(ctx)
	select {
	case err := <-b.dbRestored:
		if err != nil {
			log.Fatalf("db not restored: %v", err)
		}
	case <-b.extProp:
		// ignore (externalizations are already recorded in the state)
	}

	// start proposer who depends only on the db state (and txs in mempool)
	go b.startProposing(ctx)

	for {

		// commit to db if possible
		XXX

		// spawn more catchup routines, if necessary
		b.refillCatchup(ctx)

		select {
		case <-b.caughtUp: // catchup blocks received
		case <-b.extProp: // externalization
		}

		XXX
	}
}
