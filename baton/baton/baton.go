package baton

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func NewBaton(
	ctx context.Context,
	cfg proto.BatonConfig,
	h batonapi.Handler,
	callsPerExec int,
	batchSize int,
) *Baton {
	b := &Baton{
		ctx:             ctx,
		handler:         h,
		heartbeat:       time.Millisecond * time.Duration(cfg.Heartbeat_msec),
		callsPerExec:    callsPerExec,
		batchSize:       batchSize,
		scp:             make(chan struct{}),
		catchupRequests: map[batonapi.ID64]catchupRequest{},
		validated:       map[proto.BlockHash]validated{},
		txM:             xdrutil.NewXdrMarshaler(),
		nomM:            xdrutil.NewXdrMarshaler(),
		catchupReqM:     xdrutil.NewXdrMarshaler(),
		catchupResultM:  xdrutil.NewXdrMarshaler(),
	}
	b.cond = sync.NewCond(&b.lk)
	return b
}

type Baton struct {
	ctx          context.Context // ctx for drive
	handler      batonapi.Handler
	heartbeat    time.Duration
	callsPerExec int
	batchSize    int
	scp          chan struct{}
	cond         *sync.Cond
	lk           sync.Mutex

	// state
	committing      bool
	nominating      *nominating
	nominated       *nominated
	tip             batonapi.Tip
	catchupRequests map[batonapi.ID64]catchupRequest // pending catchup requests
	catchupBlocks   catchupBlocks                    // received catchup blocks
	externalized    []*proto.Nomination
	validated       map[proto.BlockHash]validated

	// xdr marshalers
	txM            xdrutil.XdrMarshaler
	nomM           xdrutil.XdrMarshaler
	catchupReqM    xdrutil.XdrMarshaler
	catchupResultM xdrutil.XdrMarshaler
}

type validated struct {
	*proto.Nomination
	proto.BlockID
}

func (b *Baton) lock() {
	b.lk.Lock()
}

func (b *Baton) unlock() {
	b.drive(b.ctx)
	b.lk.Unlock()
}

func (b *Baton) Kick() {
	b.lock()
	defer b.unlock()
}

func (b *Baton) Start(ctx context.Context) {
	go b.start(ctx)
}

func (b *Baton) start(ctx context.Context) {

	liblog.Infof("baton starting")

	// restore db
	var err error
	b.tip, err = b.handler.RestoreDb(ctx)
	must.Assertf(ctx, err == nil, "db not restored: %v", err)

	liblog.Infof("baton db restored")

	// release lock on scp events
	close(b.scp)

	// wake up every so often for the purpose of
	// triggering nomination which would capture any new txs that have been submitted to the mempool.
	// rationale: waking up the baton every time a tx is added to the mempool would be too expensive.
	for {
		b.Kick()
		b.handler.Sleep(b.heartbeat)
	}
}

func (b *Baton) drive(ctx context.Context) {
	defer b.cond.Broadcast()

	v := b.buildCleanView(ctx)

	// liblog.Debugf("baton stats: %s num_txs=%d num_committed_tx=%d",
	// 	v.ShortString(), b.handler.TxCount(), b.handler.CommittedTxCount())
	// liblog.Debugf("resource usage: %s", runtime.ResourceUsage())

	// validate and commit next block, if one is available and no commit is in progress
	if vb := v.NextBlockToCommit(); vb != nil && !b.committing {
		liblog.Debugf("baton committing [%v; %v]", vb.BlockHeight, form.Display(vb.BlockHash))
		b.committing = true
		// cancel ongoing nominations trumped by this commit to save compute
		b.cancelNominatingNotAbove(ctx, vb.BlockHeight)
		go func() {
			blockID, err := b.commit(ctx, vb)
			if err != nil {
				liblog.Errorf("commit: %v", err)
			}

			b.lock()
			defer b.unlock()

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
			b.purgeNotAboveTipHeight(ctx)
		}()
	}

	// start nomination, if there are no externalizations above the db tip and no nomination has started
	if tipToNominateOn := v.TipToNominateOn(); b.canNominate(tipToNominateOn, b.nominating, b.nominated) {
		// liblog.Debugf("baton nominating onto %v", tipToNominateOn)
		// cancel prior unfinished nominations
		if b.nominating != nil {
			b.nominating.Cancel()
			b.nominating = nil
		}
		nomCtx, nomCancel := context.WithCancel(ctx)
		current := &nominating{
			OnTip: *tipToNominateOn,
			Cancel: func() {
				// liblog.Debugf("baton cancelling ongoing nomination for slot %d", scpapi.Slot(tipToNominateOn.Height+1))
				nomCancel()
				// XXX: the below may be problematic
				// StopNomination is an advisory signal to stop engaging in the nomination protocol (wasting resources).
				// Stale nominations will definiteively be purged when externalizations happen.
				// b.handler.StopNomination(context.Background(), scpapi.Slot(tipToNominateOn.Height+1))
			},
		}
		b.nominating = current
		go func() {
			nomination, blockID, err := b.assembleNomination(nomCtx, *tipToNominateOn)
			current.Cancel()

			b.lock()
			defer b.unlock()

			// if nomination was cancelled, b.nominating may point at a different nomination
			currentWasReplaced := b.nominating != current
			if !currentWasReplaced {
				b.nominating = nil
			}

			if err != nil {
				if err != batonapi.ErrNothingToNominate {
					liblog.Errorf("nominate: %v", err)
				}
				return
			}

			if !currentWasReplaced {
				b.nominated = &nominated{
					Time:       b.handler.Now(),
					Nomination: nomination,
				}
				b.validated[nomination.Block_hash] = validated{Nomination: nomination, BlockID: blockID}
				liblog.Infof("nominated at block height %d", nomination.Block_height)
			}
		}()
	}

	// send catchup requests
	if needed := v.NeededUnrequestedSpans(); needed.IsNotEmpty() {
		liblog.Debugf("baton requesting catchup")
		b.requestCatchup(ctx, needed)
	}
}

func (b *Baton) WaitGrowBeyond(ctx context.Context, height proto.BlockHeight) {
	b.lk.Lock()
	defer b.lk.Unlock()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if b.seenExternalizationsAbove(height) {
			return
		}
		b.cond.Wait()
	}
}

func (b *Baton) seenExternalizationsAbove(height proto.BlockHeight) bool {
	if b.tip.Height > height {
		return true
	}
	if len(b.externalized) == 0 {
		return false
	}
	return b.externalized[len(b.externalized)-1].Block_height > height
}
