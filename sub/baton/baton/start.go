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

	// restore db
	var err error
	tipResult, err := b.handler.RestoreDb(ctx)
	if errors.Is(err, context.Canceled) {
		return
	}
	must.Assertf(ctx, err == nil, "db not restored: %v", err)
	must.Assertf(ctx, tipResult.Outcome == proto.SchedulerGetTipOK, "missing genesis block")
	b.tipDb = *tipResult.Db_tip()
	b.metrics.recordCurrentBlockHeight(ctx, protoutil.ValueHeight(&b.tipDb.Value))
	liblog.Infof(ctx, "db restored %v", protoutil.DbTipString(b.tipDb))

	// the db tip carries the consensus value that committed it;
	// the genesis block has height 0 and a zero-valued consensus value
	tipHeight := protoutil.ValueHeight(&b.tipDb.Value)

	// load known externalizations above db tip
	higherBasis, err := b.handler.CVPoolFindCommittedAbove(ctx, tipHeight)
	if errors.Is(err, context.Canceled) {
		return
	}
	must.NoError(ctx, err)
	highestExternalized := tipHeight
	for _, basis := range higherBasis {
		b.externalized = append(b.externalized, newExternalized(&basis.Value, &basis))
		if protoutil.ValueHeight(&basis.Value) > highestExternalized {
			highestExternalized = protoutil.ValueHeight(&basis.Value)
		}

		height := protoutil.ValueHeight(&basis.Value)
		block, err := b.handler.CBPoolFindCommitted(ctx, height)
		if errors.Is(err, context.Canceled) {
			return
		}
		must.NoError(ctx, err)
		if block == nil {
			continue
		}
		arrival := b.mustRestoreLocalCatchup(ctx, block, &basis)
		b.catchupArrivals = append(b.catchupArrivals, arrival)
		liblog.Infof(ctx, "local catchup restored at height %d", height)
	}

	// initialize scp
	b.handler.SCPSetLowestNonExternalized(ctx, scpkit.Slot(highestExternalized+1))
	err = b.handler.SCPReviveStateFromEnvelopes(ctx, scpkit.Slot(highestExternalized+1))
	must.NoError(ctx, err)

	// prevents scpd from processing envelopes before the initialization above completes.
	b.handler.SCPAllowFeedingEnvelopes(ctx)

	liblog.Infof(ctx, "baton db restored at height %d", tipHeight)

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

func (b *Baton) mustRestoreLocalCatchup(
	ctx context.Context,
	block *proto.CatchupBlock,
	basis *proto.ConsensusBasis,
) CatchupArrival {
	height := protoutil.ValueHeight(&basis.Value)
	proposer := block.Consensus_basis.Value.Proposal_header.Proposer
	arrival, ok := verifiedCatchupArrival(proposer, b.handler.Now(), block)
	must.Assertf(ctx, ok, "invalid local catchup block at height %d", height)
	must.Assertf(
		ctx,
		protoutil.ValueEquals(arrival.ConsensusValue(), &basis.Value),
		"local catchup block disagrees with consensus at height %d",
		height,
	)
	return arrival
}
