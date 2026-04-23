package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) Start(ctx context.Context) {
	go b.start(ctx)
	<-b.ready
}

func (b *Baton) start(ctx context.Context) {

	liblog.Infof(ctx, "baton starting")

	// restore db
	var err error
	tipResult, err := b.handler.RestoreDb(ctx)
	must.Assertf(ctx, err == nil, "db not restored: %v", err)
	must.Assertf(ctx, tipResult.Outcome == proto.SchedulerGetTipOK, "missing genesis block")
	b.tipDb = *tipResult.Db_tip()
	liblog.Infof(ctx, "db restored %v", protoutil.DbTipString(b.tipDb))

	// genesis block has height 0, and no corresponding consensus value
	if b.tipDb.Height > 0 {
		tipCV := b.handler.CVPoolFindCommitted(ctx, b.tipDb.Height)
		must.Assert(ctx, tipCV != nil)
		must.Assert(ctx, tipCV.Nomination.Block_height == b.tipDb.Height)
		b.tipExt = b.newExternalized(tipCV, b.hashCV(tipCV))
		liblog.Infof(ctx, "externalized restored (above genesis) %v", ExternalizedString(b.tipExt))
	} else {
		assumedCV := assumedConsensusValueAtHeight(b.tipDb.Height)
		b.tipExt = b.newExternalized(&assumedCV, b.hashCV(&assumedCV))
		liblog.Infof(ctx, "externalized restored (from genesis) %v", ExternalizedString(b.tipExt))
	}

	// notify scpd of the lowest non-externalized slot
	// XXX: prevent scpd from processing envelopes before this call.
	b.handler.InitSCPLowestNonExternalized(ctx, scpkit.Slot(b.tipDb.Height+1))

	// load known externalizations above db tip
	higherCV := b.handler.CVPoolFindCommittedAbove(ctx, b.tipDb.Height)
	for _, cv := range higherCV {
		b.externalized = append(b.externalized, b.newExternalized(&cv, b.hashCV(&cv)))
	}

	liblog.Infof(ctx, "baton db restored at height %d", b.tipDb.Height)

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
