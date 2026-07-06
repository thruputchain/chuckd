package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: commit currently does not handle the upgrades in the consensus value
func (b *Baton) commit(ctx context.Context, cb *batonkit.CommittableBlock) proto.BlockID {

	value := cb.ConsensusValue()
	liblog.Infof(ctx, "committing: height %d; value %s; proposer %s",
		cb.BlockHeight(), protoutil.ValueDisplay(value),
		protoutil.VIDString(cb.AsConsensusBasis().Value.Proposer))

	b.stampCommitStart(ctx, cb.BlockHeight())
	defer b.stampCommitEnd(cb.BlockHeight())

	validatedBlockID := cb.AlreadyValidated()

	if validatedBlockID == nil {
		basis := cb.AsConsensusBasis()
		proposer := basis.Value.Proposer

		if txs := cb.UnsubmittedSignedTransactions(); txs != nil {
			batch := xdrutil.ParallelizeSignedTransactionBatch(b.batchCatchupM, systemfixed.CatchupTxBatchMaxSegments, txs)
			err := b.handler.Schedulercoord_submit_vetted(ctx, proposer, batch)
			must.NoError(ctx, err)
		}

		validatedTip, err := b.validateProposal(ctx, cb.Onto, basis, cb.ConsensusValue())
		if errors.Is(err, batonkit.ErrNotValid) {
			liblog.Fatalf(ctx, "committing catchup failed to validate: %v", err)
		}
		must.NoError(ctx, err)
		validatedBlockID = &validatedTip.Id
	}

	b.handler.CVPoolCommit(ctx, *cb.AsConsensusBasis(), *cb.ConsensusValue())
	b.handler.CBPoolWaitPersist(ctx, *cb.ConsensusValue())

	outcome, err := b.handler.Schedulercoord_commit_block(ctx, *validatedBlockID, *value)
	must.NoError(ctx, err)
	must.Assertf(ctx, outcome == proto.SchedulerCommitOK, "scheduler commit outcome: %v", outcome)

	evCommitted := batonkit.EventCommitted{
		Onto:           cb.Onto,
		Height:         cb.BlockHeight(),
		ValueHash:      protoutil.HashValueToProto(value),
		DbgValidatable: cb,
	}
	liblog.Infof(ctx, "%v", evCommitted)
	b.handler.Emit(ctx, evCommitted)

	return *validatedBlockID
}

func (b *Baton) purgeNotAboveTipHeight(_ context.Context) {
	tipHeight := protoutil.TipHeight(b.tipDb)

	var catchup catchupArrivals
	for _, cb := range b.catchupArrivals {
		if cb.BlockHeight() > tipHeight {
			catchup = append(catchup, cb)
		}
	}
	b.catchupArrivals = catchup

	var exts []externalized
	for _, p := range b.externalized {
		if p.Height() > tipHeight {
			exts = append(exts, p)
		}
	}
	b.externalized = exts

	for h, p := range b.validated {
		if p.ConsensusValue.Proposal_header.Block_height <= tipHeight {
			delete(b.validated, h)
		}
	}

	for h, p := range b.invalidated {
		if p.ConsensusBasis.Value.Proposal_header.Block_height <= tipHeight {
			delete(b.invalidated, h)
		}
	}
}
