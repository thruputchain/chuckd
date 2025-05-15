package baton

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func NewBaton(ctx context.Context, h Handler) *Baton {
	return &Baton{
		handler:         h,
		scp:             make(chan struct{}),
		catchupRequests: map[ID64]catchupRequest{},
		validated:       map[proto.BlockHash]validated{},
	}
}

type Baton struct {
	handler Handler
	scp     chan struct{}
	lk      sync.Mutex
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

func (b *Baton) lock(_ context.Context) {
	b.lk.Lock()
}

func (b *Baton) unlock(ctx context.Context) {
	b.drive(ctx)
	b.lk.Unlock()
}

// Kick must be called after new transactions or batches are inserted into the mempools.
func (b *Baton) Kick(ctx context.Context) {
	b.lock(ctx)
	defer b.unlock(ctx)
}

func (b *Baton) Start(ctx context.Context) {
	go b.start(ctx)
}

func (b *Baton) start(ctx context.Context) {

	// restore db
	var err error
	b.tip, err = b.handler.RestoreDb(ctx)
	must.Assertf(ctx, err == nil, "db not restored: %v", err)

	b.Kick(ctx)
	for {
		b.scp <- struct{}{}
	}
}

func (b *Baton) drive(ctx context.Context) {
	// build view
	v := b.buildCleanView(ctx)
	// log.Debugf("view %v", form.SprintJSON(v))

	// validate and commit next block, if one is available and no commit is in progress
	if vb := v.NextBlockToCommit(); vb != nil && !b.committing {
		log.Debugf("committing [%v; %v]", vb.BlockHeight, form.SprintJSON(vb.BlockHash))
		b.committing = true
		go func() {
			blockID, err := b.commit(ctx, vb)
			if err != nil {
				log.Errorf("commit: %v", err)
			}

			b.lock(ctx)
			defer b.unlock(ctx)

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
			nomCtx, nomCancel := context.WithCancel(ctx)
			b.nominating = &nominating{Tip: *nominate, CancelFunc: nomCancel}
			go func() {
				proposal, blockID, err := b.nominate(nomCtx, *nominate, uncommittedTxs)
				if err != nil {
					log.Errorf("nominate: %v", err)
				}

				b.lock(ctx)
				defer b.unlock(ctx)

				must.Assertf(ctx, b.nominating != nil, "race nominating")
				b.nominating.CancelFunc()
				b.nominating = nil

				if err != nil { // nomination failed
					return
				}

				b.nominated = &nominated{Proposal: proposal}
				b.validated[proposal.Block_hash] = validated{Proposal: proposal, BlockID: blockID}
			}()
		}
	}

	// send catchup requests
	if needed := v.NeededUnrequestedSpans(); needed.IsNotEmpty() {
		b.requestCatchup(ctx, needed)
	}
}
