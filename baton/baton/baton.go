package baton

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func NewBaton(ctx context.Context, h sdk.Handler) *Baton {
	b := &Baton{
		handler:         h,
		scp:             make(chan struct{}),
		catchupRequests: map[sdk.ID64]catchupRequest{},
		validated:       map[proto.BlockHash]validated{},
	}
	b.cond = sync.NewCond(&b.lk)
	return b
}

type Baton struct {
	handler sdk.Handler
	scp     chan struct{}
	cond    *sync.Cond
	lk      sync.Mutex
	// state
	committing      bool
	nominating      *nominating
	nominated       *nominated
	tip             sdk.Tip
	catchupRequests map[sdk.ID64]catchupRequest // pending catchup requests
	catchupBlocks   catchupBlocks               // received catchup blocks
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

func (b *Baton) Start(ctx context.Context, heartbeat time.Duration) {
	go b.start(ctx, heartbeat)
}

func (b *Baton) start(ctx context.Context, heartbeat time.Duration) {

	log.Infof("baton starting")

	// restore db
	var err error
	b.tip, err = b.handler.RestoreDb(ctx)
	must.Assertf(ctx, err == nil, "db not restored: %v", err)

	log.Infof("baton db restored")

	// release lock on scp events
	close(b.scp)

	// wake up every so often for the purpose of
	// triggering nomination which would capture any new txs that have been submitted to the mempool.
	// rationale: waking up the baton every time a tx is added to the mempool would be too expensive.
	for {
		b.Kick(ctx)
		b.handler.Sleep(heartbeat)
	}
}

func (b *Baton) drive(ctx context.Context) {
	defer b.cond.Broadcast()

	// log.Infof("baton driving: num_txs=%d num_batches=%d num_cmmitted_tx=%d",
	// 	b.handler.TxCount(), b.handler.BatchCount(), b.handler.CommittedTxCount())

	v := b.buildCleanView(ctx)

	// validate and commit next block, if one is available and no commit is in progress
	if vb := v.NextBlockToCommit(); vb != nil && !b.committing {
		log.Debugf("baton committing [%v; %v]", vb.BlockHeight, form.Display(vb.BlockHash))
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
		log.Debugf("baton nominating onto %v", nominate)
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
		log.Debugf("baton requesting catchup")
		b.requestCatchup(ctx, needed)
	}
}

func (b *Baton) WaitTipOrConsensusGrow(ctx context.Context, height proto.BlockHeight) {
	b.lk.Lock()
	defer b.lk.Unlock()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if b.tip.Height > height {
			return
		}

		b.cond.Wait()
	}
}
