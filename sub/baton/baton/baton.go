package baton

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func NewBaton(
	ctx context.Context,
	cfg proto.BatonConfig,
	h batonkit.Handler,
) *Baton {
	b := &Baton{
		ctx:              ctx,
		handler:          h,
		skew:             time.Millisecond * time.Duration(cfg.Skew_msec),
		heartbeat:        time.Millisecond * time.Duration(cfg.Heartbeat_msec),
		patience:         time.Millisecond * time.Duration(cfg.Patience_msec),
		maxCatchupHeight: int(cfg.Max_catchup_height),
		ready:            make(chan struct{}),
		catchupWaiting:   map[batonkit.ID64]catchupRequest{},
		validated:        map[proto.ConsensusValueHash]validated{},
		invalidated:      map[proto.ConsensusValueHash]invalidated{},
		cvM:              xdrutil.NewXdrMarshaler(),
		txM:              xdrutil.NewXdrMarshaler(),
		nomM:             xdrutil.NewXdrMarshaler(),
		catchupReqM:      xdrutil.NewXdrMarshaler(),
		catchupResultM:   xdrutil.NewXdrMarshaler(),
	}
	b.cond = sync.NewCond(&b.lk)
	return b
}

type Baton struct {
	ctx              context.Context // ctx for drive
	handler          batonkit.Handler
	skew             time.Duration
	heartbeat        time.Duration
	patience         time.Duration // if no externalization seen for this duration, request a backfeed
	maxCatchupHeight int
	ready            chan struct{}
	cond             *sync.Cond
	lk               sync.Mutex

	// state, read/write under lk
	committing      bool
	nominating      *nominating
	nominated       *nominated
	tipDb           proto.DbTip
	tipExt          externalizedConsensus
	catchupWaiting  map[batonkit.ID64]catchupRequest // pending catchup requests
	catchupArrivals catchupArrivals                  // received catchup blocks
	externalized    []externalized                   // known externalizations
	validated       map[proto.ConsensusValueHash]validated
	invalidated     map[proto.ConsensusValueHash]invalidated
	liveness        liveness

	// xdr marshalers
	cvM            xdrutil.XdrMarshaler // for marshalling ConsensusValue
	txM            xdrutil.XdrMarshaler // for marshalling SignedTransaction
	nomM           xdrutil.XdrMarshaler
	catchupReqM    xdrutil.XdrMarshaler
	catchupResultM xdrutil.XdrMarshaler
}

type validated struct {
	*proto.ConsensusValue
	proto.BlockID
}

type invalidated struct {
	*proto.ConsensusValue
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
	<-b.ready
}

func (b *Baton) start(ctx context.Context) {

	liblog.Infof("baton starting")

	// restore db
	var err error
	b.tipDb, err = b.handler.RestoreDb(ctx)
	must.Assertf(ctx, err == nil, "db not restored: %v", err)
	must.Assertf(ctx, b.tipDb.Height > 0, "db tip height 0; missing genesis block")

	// genesis block has height 1, and no corresponding consensus value
	if b.tipDb.Height > 1 {
		tipCV := b.handler.FindConsensusValue(ctx, b.tipDb.Height)
		must.Assert(ctx, tipCV != nil)
		must.Assert(ctx, tipCV.Nomination.Block_height == b.tipDb.Height)
		b.tipExt = b.newExternalized(tipCV)
	} else {
		acv := assumedConsensusValueAtHeight(b.tipDb.Height)
		b.tipExt = b.newExternalized(&acv)
	}

	// load known externalizations above db tip
	higherCV := storage.FindConsensusValuesAbove(ctx, b.handler, b.tipDb.Height)
	for _, cv := range higherCV {
		b.externalized = append(b.externalized, b.newExternalized(&cv))
	}

	liblog.Infof("baton db restored at height %d", b.tipDb.Height)

	// release lock on scp events
	close(b.ready)

	b.lock()
	b.noteStarted()
	b.unlock()

	// wake up every so often for the purpose of
	// triggering nomination which would capture any new txs that have been submitted to the mempool.
	// rationale: waking up the baton every time a tx is added to the mempool would be too expensive.
	for {
		b.handler.Sleep(b.heartbeat)
		b.Kick()
	}
}

func assumedConsensusValueAtHeight(height proto.BlockHeight) proto.ConsensusValue {
	return proto.ConsensusValue{Nomination: proto.Nomination{Block_height: height}}
}

func (b *Baton) WaitExternalizedBeyond(ctx context.Context, height proto.BlockHeight) {
	b.lk.Lock()
	defer b.lk.Unlock()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if b.seenExternalizationsBeyond(height) {
				return
			}
			b.cond.Wait()
		}
	}
}

func (b *Baton) seenExternalizationsBeyond(height proto.BlockHeight) bool {
	if b.tipDb.Height >= height {
		return true
	}
	if len(b.externalized) == 0 {
		return false
	}
	return b.externalized[len(b.externalized)-1].Height() >= height
}

var ErrTipAlreadyAhead = fmt.Errorf("tip already ahead")

func (b *Baton) WaitCommittedAt(ctx context.Context, height proto.BlockHeight) (*proto.DbTip, *externalizedConsensus, error) {
	liblog.Infof("waiting for commit at height %d", height)

	b.lk.Lock()
	defer b.lk.Unlock()
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
			switch {
			case b.tipDb.Height > height:
				return nil, nil, ErrTipAlreadyAhead
			case b.tipDb.Height == height:
				copyTipDb, copyTipExt := b.tipDb, b.tipExt
				return &copyTipDb, &copyTipExt, nil
			default:
				b.cond.Wait()
			}
		}
	}
}
