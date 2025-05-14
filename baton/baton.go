package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func NewBaton(ctx context.Context, h Handler) *Baton {
	return &Baton{
		handler:         h,
		synchronize:     make(chan synchronize, 1),
		catchupRequests: map[ID64]catchupRequest{},
		validated:       map[proto.BlockHash]validated{},
	}
}

func (b *Baton) Start(ctx context.Context) {
	go b.startBaton(ctx)
}

type Baton struct {
	handler     Handler
	synchronize chan synchronize
	// state
	committing      bool
	nominating      context.CancelFunc
	tip             Tip
	catchupRequests map[ID64]catchupRequest // pending catchup requests
	catchupBlocks   catchupBlocks           // received catchup blocks
	externalized    []*proto.Proposal
	validated       map[proto.BlockHash]validated
}

type synchronize func()

func (b *Baton) startBaton(ctx context.Context) {

	// synchronize chan is active right away with potential calls from scp

	// restore db
	restored := make(chan synchronize)
	go b.restoreDb(ctx, restored)
WaitDb:
	for {
		select {
		case restored := <-restored:
			restored()
			break WaitDb
		case <-b.synchronize:
			// ignore scp (externalized or validate value) until db is restored
		}
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

					if b.tip.Height+1 != vb.BlockHeight {
						panic("bug")
					}

					b.tip.Hash = vb.BlockHash
					b.tip.Height = vb.BlockHeight
					b.tip.ID = blockID

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

					b.validated[proposal.Block_hash] = validated{Proposal: proposal, BlockID: blockID}
				}
			}()
		}

		// send catchup requests
		if needed := v.NeededUnrequestedSpans(); needed.IsNotEmpty() {
			b.requestCatchup(ctx, needed)
		}

		// wait until something happens
		select {
		case synchronize := <-b.synchronize:
			synchronize()
		}
	}
}
