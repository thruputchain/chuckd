package baton

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblock"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

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
	committing      *proto.BlockHeight
	nominating      *proposing
	tipDb           proto.DbTip
	catchupWaiting  map[batonkit.ID64]catchupRequest // pending catchup requests
	catchupArrivals catchupArrivals                  // received catchup blocks
	externalized    []externalized                   // known externalizations
	validating      *liblock.KeyedMutex[protoutil.ValueHash]
	validated       map[protoutil.ValueHash]validated
	invalidated     map[protoutil.ValueHash]invalidated
	liveness        liveness

	// observability
	metrics batonMetrics
	chrono  chrono

	// xdr marshalers (no lock necessary)
	cvM            libxdr.XdrMarshaler // for marshalling ConsensusValue
	nomM           libxdr.XdrMarshaler
	catchupReqM    libxdr.XdrMarshaler
	catchupResultM libxdr.XdrMarshaler
	batchCatchupM  libxdr.XdrMarshaler
	batchRepairM   libxdr.XdrMarshaler
}

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
		validating:       liblock.NewKeyedMutex[protoutil.ValueHash](),
		validated:        map[protoutil.ValueHash]validated{},
		invalidated:      map[protoutil.ValueHash]invalidated{},
		cvM:              libxdr.NewXdrMarshaler(),
		nomM:             libxdr.NewXdrMarshaler(),
		catchupReqM:      libxdr.NewXdrMarshaler(),
		catchupResultM:   libxdr.NewXdrMarshaler(),
		batchCatchupM:    libxdr.NewXdrMarshaler(),
		batchRepairM:     libxdr.NewXdrMarshaler(),
		metrics:          newBatonMetrics(ctx),
	}
	b.cond = sync.NewCond(&b.lk)
	return b
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

func (b *Baton) WaitExternalizedBeyond(ctx context.Context, height proto.BlockHeight) {
	stop := context.AfterFunc(ctx, func() {
		b.lk.Lock()
		defer b.lk.Unlock()
		b.cond.Broadcast()
	})
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

// tipHeight returns the height of the db tip. Must be called under lk.
func (b *Baton) tipHeight() proto.BlockHeight {
	return protoutil.ValueHeight(&b.tipDb.Value)
}

func (b *Baton) seenExternalizationsBeyond(height proto.BlockHeight) bool {
	if b.tipHeight() >= height {
		return true
	}
	if len(b.externalized) == 0 {
		return false
	}
	return b.externalized[len(b.externalized)-1].Height() >= height
}

func (b *Baton) HighestCommitted() proto.BlockHeight {
	b.lk.Lock()
	defer b.lk.Unlock()

	return b.tipHeight()
}

func (b *Baton) HighestCommittedOrExternalized() proto.BlockHeight {
	b.lk.Lock()
	defer b.lk.Unlock()

	if len(b.externalized) == 0 {
		return b.tipHeight()
	}
	return b.externalized[len(b.externalized)-1].Height()
}

var ErrTipAlreadyAhead = fmt.Errorf("tip already ahead")

func (b *Baton) WaitCommittedAt(ctx context.Context, height proto.BlockHeight) (*proto.DbTip, error) {
	liblog.Infof(ctx, "waiting for commit at height %d", height)

	stop := context.AfterFunc(ctx, func() {
		b.lk.Lock()
		defer b.lk.Unlock()
		b.cond.Broadcast()
	})
	defer stop()

	b.lk.Lock()
	defer b.lk.Unlock()
	for {
		select {
		case <-ctx.Done():
			liblog.Errorf(ctx, "waiting for commit at height %d: context cancelled", height)
			return nil, ctx.Err()
		default:
			switch {
			case b.tipHeight() > height:
				liblog.Errorf(ctx, "waiting for commit at height %d: tip already ahead", height)
				return nil, ErrTipAlreadyAhead
			case b.tipHeight() == height:
				liblog.Infof(ctx, "waiting for commit at height %d: tip reached", height)
				copyTipDb := b.tipDb
				return &copyTipDb, nil
			default:
				b.cond.Wait()
			}
		}
	}
}
