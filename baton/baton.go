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
		dbRestored: make(chan dbRestored, 1),
		extProp:    make(chan extProp),
		caughtUp:   make(chan caughtUp),
	}
	b.catchup.pending = map[proto.ValidatorID]*catchupPending{}
	b.catchup.blocks = map[proto.ValidatorID]*catchupBlocks{}

	go b.startBaton(ctx)
	// start mempool
	// ...
	return b
}

type Baton struct {
	handler Handler
	// callbacks
	dbRestored chan dbRestored
	extProp    chan extProp
	caughtUp   chan caughtUp
	// states
	db struct { // local db
		lk        sync.Mutex
		tipHeight proto.BlockHeight
		tipHash   proto.BlockHash
	}
	catchup struct { // catching up
		lk      sync.Mutex
		pending map[proto.ValidatorID]*catchupRequests // pending catchup requests
		blocks  map[proto.ValidatorID]*catchupBlocks   // received catchup blocks
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

	// extProp is active right away

	// restore db
	go b.restoreDb(ctx)
	select {
	case r := <-b.dbRestored:
		if r.Err != nil {
			log.Fatalf("db not restored: %v", r.Err)
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
