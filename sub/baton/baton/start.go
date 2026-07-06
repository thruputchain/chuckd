package baton

import (
	"context"
	"errors"

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

	var err error
	tipResult, err := b.handler.RestoreDb(ctx)
	if errors.Is(err, context.Canceled) {
		return
	}
	must.Assertf(ctx, err == nil, "db not restored: %v", err)
	must.Assertf(ctx, tipResult.Outcome == proto.SchedulerGetTipOK, "missing genesis block")
	b.tipDb = *tipResult.Db_tip()
	liblog.Infof(ctx, "db restored %v", protoutil.DbTipString(b.tipDb))

	tipValue := b.tipDb.Value
	tipBasis := proto.ConsensusBasis{Value: tipValue}
	b.tipExt = newExternalized(&tipValue, &tipBasis)
	liblog.Infof(ctx, "externalized restored %v", ExternalizedString(b.tipExt))

	higherBasis, err := b.handler.CVPoolFindCommittedAbove(ctx, protoutil.TipHeight(b.tipDb))
	if errors.Is(err, context.Canceled) {
		return
	}
	must.NoError(ctx, err)
	highestExternalized := protoutil.TipHeight(b.tipDb)
	for _, basis := range higherBasis {
		b.externalized = append(b.externalized, newExternalized(&basis.Value, &basis))
		if basis.Value.Proposal_header.Block_height > highestExternalized {
			highestExternalized = basis.Value.Proposal_header.Block_height
		}
	}

	b.handler.SCPSetLowestNonExternalized(ctx, scpkit.Slot(highestExternalized+1))
	err = b.handler.SCPReviveStateFromEnvelopes(ctx, scpkit.Slot(highestExternalized+1))
	must.NoError(ctx, err)

	b.handler.SCPAllowFeedingEnvelopes(ctx)

	liblog.Infof(ctx, "baton db restored at height %d", protoutil.TipHeight(b.tipDb))

	close(b.ready)

	b.lock()
	b.noteStarted()
	b.unlock()

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
