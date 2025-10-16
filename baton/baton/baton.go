package baton

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
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
		heartbeat:        time.Millisecond * time.Duration(cfg.Heartbeat_msec),
		maxCatchupHeight: int(cfg.Max_catchup_height),
		scp:              make(chan struct{}),
		catchupRequests:  map[batonkit.ID64]catchupRequest{},
		validated:        map[proto.BlockHash]validated{},
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
	heartbeat        time.Duration
	maxCatchupHeight int
	scp              chan struct{}
	cond             *sync.Cond
	lk               sync.Mutex

	// state
	committing      bool
	nominating      *nominating
	nominated       *nominated
	tip             proto.DbTip
	catchupRequests map[batonkit.ID64]catchupRequest // pending catchup requests
	catchupBlocks   catchupBlocks                    // received catchup blocks
	externalized    []*proto.ConsensusValue
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
	if b.tip.Height >= height {
		return true
	}
	if len(b.externalized) == 0 {
		return false
	}
	return b.externalized[len(b.externalized)-1].Nomination.Block_height >= height
}

var ErrTipAlreadyAhead = fmt.Errorf("tip already ahead")

func (b *Baton) WaitCommittedAt(ctx context.Context, height proto.BlockHeight) (proto.DbTip, error) {
	b.lk.Lock()
	defer b.lk.Unlock()
	for {
		select {
		case <-ctx.Done():
			return proto.DbTip{}, ctx.Err()
		default:
			switch {
			case b.tip.Height > height:
				return proto.DbTip{}, ErrTipAlreadyAhead
			case b.tip.Height == height:
				return b.tip, nil
			default:
				b.cond.Wait()
			}
		}
	}
}
