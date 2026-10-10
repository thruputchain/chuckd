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
	"go.opentelemetry.io/otel/attribute"
)

// XXX: commit currently does not handle the upgrades in the consensus value
func (b *Baton) commit(ctx context.Context, cb *batonkit.CommittableBlock) proto.BlockID {

	value := cb.ConsensusValue()
	liblog.Infof(ctx, "committing: height %d; value %s; proposer %s",
		cb.BlockHeight(),
		protoutil.ValueDisplay(value),
		protoutil.VIDString(protoutil.ValueProposer(value)))

	b.stampCommitStart(ctx, cb.BlockHeight())
	b.chrono.lk.Lock()
	if x := b.chrono.existing(cb.BlockHeight()); x != nil {
		x.attributes = []attribute.KeyValue{
			attribute.Int("tx_count", cb.NumTransactions()),
			attribute.String("proposer", protoutil.VIDString(protoutil.ValueProposer(value))),
			attribute.String("value.hash_prefix", protoutil.ValueDisplay(value)),
		}
		x.networkHash, x.hasNetworkHash = value.Proposal_header.Network_hash, true
	}
	b.chrono.lk.Unlock()
	defer b.stampCommitEnd(cb.BlockHeight())

	validatedBlockID := cb.AlreadyValidated()

	// validate, if not already validated
	if validatedBlockID == nil {

		basis := cb.AsConsensusBasis()
		proposer := basis.Value.Proposal_header.Proposer

		if txs := cb.UnsubmittedSignedTransactions(); txs != nil {
			// submit catchup transactions
			batch := xdrutil.ParallelizeSignedTransactionBatch(b.batchCatchupM, systemfixed.CatchupTxBatchMaxSegments, txs)
			err := b.handler.Schedulercoord_submit_vetted(ctx, proposer, batch)
			must.NoError(ctx, err)
		}

		blockID, err := b.validateProposal(ctx, cb.Onto, basis, cb.ConsensusValue())
		// an error cannot arise here:
		// - context cancellation or deadlines do not apply on the commit path
		// - ErrNotValid is not expected on the commit path,
		// as the block being validated arises from an externalization or
		// catchup (whose hash has been linked to a following externalization)
		if errors.Is(err, batonkit.ErrNotValid) {
			// if the catchup block is verified (which it is), this should never happen
			// unless a quorum of validators maliciously externalized an invalid block.
			liblog.Fatalf(ctx, "committing catchup failed to validate: %v", err)
		}
		must.NoError(ctx, err)
		validatedBlockID = &blockID
	}

	// CV must be guaranteed persisted before the db commits.
	b.handler.CVPoolCommit(ctx, *cb.AsConsensusBasis(), *cb.ConsensusValue())

	// catchup block must be guaranteed persisted before the db commits.
	b.handler.CBPoolWaitPersist(ctx, *cb.ConsensusValue())

	// commit to db
	// on commit the block is associated with the consensus value that confirmed it
	outcome, err := b.handler.Schedulercoord_commit_block(ctx, *validatedBlockID, *cb.ConsensusValue())
	must.NoError(ctx, err)
	must.Assertf(ctx, outcome == proto.SchedulerCommitOK, "scheduler commit outcome: %v", outcome)

	evCommitted := batonkit.EventCommitted{
		Onto:           cb.Onto,
		Height:         cb.BlockHeight(),
		DbgValidatable: cb,
	}
	liblog.Infof(ctx, "%v", evCommitted)
	b.handler.Emit(ctx, evCommitted)

	return *validatedBlockID
}

func (b *Baton) purgeNotAboveTipHeight(_ context.Context) {

	tipHeight := protoutil.ValueHeight(&b.tipDb.Value)

	// purge catchup blocks
	var catchup catchupArrivals
	for _, cb := range b.catchupArrivals {
		if cb.BlockHeight() > tipHeight {
			catchup = append(catchup, cb)
		}
	}
	b.catchupArrivals = catchup

	// purge externalized consensus values
	var exts []externalized
	for _, p := range b.externalized {
		if p.Height() > tipHeight {
			exts = append(exts, p)
		}
	}
	b.externalized = exts

	// purge validations
	for h, p := range b.validated {
		if protoutil.ValueHeight(&p.ConsensusBasis.Value) <= tipHeight {
			delete(b.validated, h)
		}
	}

	// purge invalidations
	for h, p := range b.invalidated {
		if protoutil.ValueHeight(&p.ConsensusBasis.Value) <= tipHeight {
			delete(b.invalidated, h)
		}
	}
}
