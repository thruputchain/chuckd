package baton

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblock"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func NewBaton(
	ctx context.Context,
	cfg proto.BatonConfig,
	handler batonkit.Handler,
) *Baton {
	b := &Baton{
		ctx:              ctx,
		handler:          handler,
		skew:             time.Millisecond * time.Duration(cfg.Skew_msec),
		heartbeat:        time.Millisecond * time.Duration(cfg.Heartbeat_msec),
		patience:         time.Millisecond * time.Duration(cfg.Patience_msec),
		maxCatchupHeight: int(cfg.Max_catchup_height),
		ready:            make(chan struct{}),
		catchupWaiting:   map[batonkit.ID64]catchupRequest{},
		validating:       liblock.NewKeyedMutex[proto.ConsensusValueHash](),
		validated:        map[proto.ConsensusValueHash]validated{},
		invalidated:      map[proto.ConsensusValueHash]invalidated{},
		cvM:              libxdr.NewXdrMarshaler(),
		nomM:             libxdr.NewXdrMarshaler(),
		catchupReqM:      libxdr.NewXdrMarshaler(),
		catchupResultM:   libxdr.NewXdrMarshaler(),
		batchCatchupM:    libxdr.NewXdrMarshaler(),
		batchRepairM:     libxdr.NewXdrMarshaler(),
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

	// main lock
	cond *sync.Cond
	lk   sync.Mutex

	// hooks and levers
	hooks  batonkit.BatonHooks
	levers batonkit.BatonLevers

	// state, under lk
	committing      bool
	nominating      *nominating
	tipDb           proto.DbTip
	tipExt          externalizedConsensus
	catchupWaiting  map[batonkit.ID64]catchupRequest // pending catchup requests
	catchupArrivals catchupArrivals                  // received catchup blocks
	externalized    []externalized                   // known externalizations
	validating      *liblock.KeyedMutex[proto.ConsensusValueHash]
	validated       map[proto.ConsensusValueHash]validated
	invalidated     map[proto.ConsensusValueHash]invalidated
	liveness        liveness

	// metrics (no lock necessary)
	chrono chrono

	// xdr marshalers (no lock necessary)
	cvM            libxdr.XdrMarshaler // for marshalling ConsensusValue
	nomM           libxdr.XdrMarshaler
	catchupReqM    libxdr.XdrMarshaler
	catchupResultM libxdr.XdrMarshaler
	batchCatchupM  libxdr.XdrMarshaler
	batchRepairM   libxdr.XdrMarshaler
}

type validated struct {
	*proto.ConsensusValue
	proto.BlockID
}

type invalidated struct {
	*proto.ConsensusValue
}

func (b *Baton) SetLevers(levers batonkit.BatonLevers) {
	b.lockNoDrive()
	defer b.unlockNoDrive()
	b.levers = levers
}

func (b *Baton) GetLevers() batonkit.BatonLevers {
	b.lockNoDrive()
	defer b.unlockNoDrive()
	return b.levers
}

func (b *Baton) SetHooks(hooks batonkit.BatonHooks) {
	b.lockNoDrive()
	defer b.unlockNoDrive()
	b.hooks = hooks
}

func (b *Baton) PauseNominating() {
	b.lockNoDrive()
	defer b.unlockNoDrive()
	b.levers.PauseNominating = true
}

func (b *Baton) ResumeNominating() {
	b.lockNoDrive()
	defer b.unlockNoDrive()
	b.levers.PauseNominating = false
}

func (b *Baton) lockNoDrive() {
	b.lk.Lock()
}

func (b *Baton) unlockNoDrive() {
	b.lk.Unlock()
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
	tipResult, err := b.handler.RestoreDb(ctx)
	must.Assertf(ctx, err == nil, "db not restored: %v", err)
	must.Assertf(ctx, tipResult.Outcome == proto.SchedulerGetTipOK, "missing genesis block")
	b.tipDb = *tipResult.Db_tip()
	liblog.Infof("db restored %v", protoutil.DbTipString(b.tipDb))

	// genesis block has height 0, and no corresponding consensus value
	if b.tipDb.Height > 0 {
		tipCV := b.handler.ConsensusValuePoolFindHeight(ctx, b.tipDb.Height)
		must.Assert(ctx, tipCV != nil)
		must.Assert(ctx, tipCV.Nomination.Block_height == b.tipDb.Height)
		b.tipExt = b.newExternalized(tipCV, b.hashCV(tipCV))
		liblog.Infof("externalized restored (above genesis) %v", ExternalizedString(b.tipExt))
	} else {
		assumedCV := assumedConsensusValueAtHeight(b.tipDb.Height)
		b.tipExt = b.newExternalized(&assumedCV, b.hashCV(&assumedCV))
		liblog.Infof("externalized restored (from genesis) %v", ExternalizedString(b.tipExt))
	}

	// notify scpd of the lowest non-externalized slot
	b.handler.SetLowestNonExternalized(ctx, scpkit.Slot(b.tipDb.Height+1))

	// load known externalizations above db tip
	higherCV := b.handler.ConsensusValuePoolFindAbove(ctx, b.tipDb.Height)
	for _, cv := range higherCV {
		b.externalized = append(b.externalized, b.newExternalized(&cv, b.hashCV(&cv)))
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
		select {
		case <-ctx.Done():
			return
		default:
		}
		b.handler.Sleep(b.heartbeat)
		b.Kick()
	}
}

func assumedConsensusValueAtHeight(height proto.BlockHeight) proto.ConsensusValue {
	return proto.ConsensusValue{Nomination: proto.Nomination{Block_height: height}}
}

func (b *Baton) WaitExternalizedBeyond(ctx context.Context, height proto.BlockHeight) {
	stop := context.AfterFunc(ctx, func() { b.cond.Broadcast() })
	defer stop()
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

	stop := context.AfterFunc(ctx, func() { b.cond.Broadcast() })
	defer stop()
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
