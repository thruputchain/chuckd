package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/form"
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
	nominating      *nominating
	nominated       *nominated
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
		log.Debugf("baton loop")

		// build view
		v := b.buildCleanView(ctx)

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
		if nominate := v.ReadyToNominate(); canNominate(nominate, b.nominating, b.nominated) {
			// cancel prior unfinished nominations
			if b.nominating != nil {
				b.nominating.CancelFunc()
				b.nominating = nil
			}
			uncommittedTxs := b.gatherUncommittedTxs(ctx)
			if len(uncommittedTxs) > 0 {
				log.Debugf("start nominating onto %v", form.SprintJSON(nominate))
				nomCtx, nomCancel := context.WithCancel(ctx)
				b.nominating = &nominating{Tip: *nominate, CancelFunc: nomCancel}
				go func() {
					proposal, blockID, err := b.nominate(nomCtx, *nominate, uncommittedTxs)
					if err != nil {
						log.Errorf("nominate: %v", err)
					}
					b.synchronize <- func() {
						log.Debugf("end nominating onto %v", form.SprintJSON(nominate))
						must.Assertf(ctx, b.nominating != nil, "race nominating")
						b.nominating.CancelFunc()
						b.nominating = nil

						if err != nil { // nomination failed
							return
						}

						b.nominated = &nominated{Proposal: proposal}
						b.validated[proposal.Block_hash] = validated{Proposal: proposal, BlockID: blockID}
					}
				}()
			}
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
