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
		handler:     h,
		dbRestored:  make(chan dbRestored, 1),
		synchronize: make(chan synchronize, 1),
		caughtUp:    make(chan caughtUp),
	}
	b.catchup.pending = map[ID64]catchupRequest{}
	b.catchup.blocks = nil
	b.consensus.validated = map[proto.BlockHash]validated{}
	return b
}

func (b *Baton) Start(ctx context.Context) {
	go b.startBaton(ctx)
}

type Baton struct {
	handler Handler
	// notifications to baton
	dbRestored  chan dbRestored
	synchronize chan synchronize
	caughtUp    chan caughtUp
	// state
	committing bool
	nominating context.CancelFunc
	db         struct {
		lk  sync.Mutex
		tip Tip
	}
	catchup struct {
		lk      sync.Mutex
		pending map[ID64]catchupRequest // pending catchup requests
		blocks  catchupBlocks           // received catchup blocks
	}
	consensus struct {
		ext       []*proto.Proposal
		validated map[proto.BlockHash]validated
	}
}

type synchronize func()

func (b *Baton) startBaton(ctx context.Context) {

	// synchronize chan is active right away

	// restore db
	go b.restoreDb(ctx)
	select {
	case r := <-b.dbRestored:
		if r.Err != nil {
			log.Fatalf("db not restored: %v", r.Err)
		}
		log.Debugf("<-b.dbRestored")
	case <-b.synchronize:
		// XXX: ignore ???
	}

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

		// validate and commit next block, if one is available and no commit is in progress
		if vb := v.NextBlockToCommit(); vb != nil && !b.committing {
			b.committing = true
			go func() {
				blockID, err := b.commit(ctx, vb)
				if err != nil {
					log.Errorf("commit: %v", err)
				}
				b.synchronize <- func() {
					must.Assertf(ctx, b.committing, "race committing")
					b.committing = false

					if err != nil { // commit failed
						return
					}

					if b.db.tip.Height+1 != vb.BlockHeight {
						panic("bug")
					}

					b.db.tip.Hash = vb.BlockHash
					b.db.tip.Height = vb.BlockHeight
					b.db.tip.ID = blockID

					// purge catchup or consensus blocks no higher than the newly committed one
					b.purgeBelowTip(ctx)
				}
			}()
		}

		// start nomination, if the db there are no externalizations above the db tip
		if nom := v.ReadyToNominate(); nom != nil {
			// cancel prior unfinished nominations
			if b.nominating != nil {
				b.nominating()
				b.nominating = nil
			}
			nomCtx, nomCancel := context.WithCancel(ctx)
			b.nominating = nomCancel
			go func() {
				proposal, blockID, err := b.nominate(nomCtx, nom.tip)
				if err != nil {
					log.Errorf("nominate: %v", err)
				}
				b.synchronize <- func() {
					must.Assertf(ctx, b.nominating != nil, "race nominating")
					b.nominating = nil

					if err != nil { // nomination failed
						return
					}

					b.consensus.validated[proposal.Block_hash] = validated{Proposal: proposal, BlockID: blockID}
				}
			}()
		}

		// send catchup requests
		if needed := v.NeededUnrequestedSpans(); needed.IsNotEmpty() {
			XXX // next up to "synchronize"
			b.requestCatchup(ctx, needed)
		}

		// wait until something happens
		select {
		case synchronize := <-b.synchronize:
			synchronize()
		case <-b.caughtUp: // catchup blocks received
			log.Debugf("<-b.caughtUp")
		}
	}
}
