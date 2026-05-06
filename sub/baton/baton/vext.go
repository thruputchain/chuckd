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

func (b *Baton) validateExternalizedCue(
	ctx context.Context,
	slot scpkit.Slot,
	cvc *proto.ConsensusValueCue,
	reply chan<- scpkit.ValidationLevel,
) {
	liblog.Debugf(ctx, "validating cue for already externalized slot %d", slot)

	cv, err := b.handler.CVPoolFindCommitted(ctx, proto.BlockHeight(slot))
	if errors.Is(err, context.Canceled) {
		// context cancellation is expected only if validation is being aborted due to externalizations ahead.
		// cancellation on this condition is triggered in validateCue.
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= proto.BlockHeight(slot))
		reply <- scpkit.MaybeValidValue
		return
	}
	must.NoError(ctx, err)
	must.Assert(ctx, cv != nil)

	if protoutil.CVAgreesWithCue(cv, cvc) {
		reply <- scpkit.FullyValidValue
	} else {
		reply <- scpkit.InvalidValue
	}
}

// validateCommittedConsensus validates a consensus value that has been committed to the database.
func (b *Baton) validateCommittedConsensus(
	ctx context.Context,
	slot scpkit.Slot,
	_ *proto.ConsensusValue,
	cvc *proto.ConsensusValueCue,
	reply chan<- scpkit.ValidationLevel,
) {
	liblog.Debugf(ctx, "validating consensus value for already externalized slot %d", slot)

	cvx, err := b.handler.CVPoolFindCommitted(ctx, proto.BlockHeight(slot))
	if errors.Is(err, context.Canceled) {
		// context cancellation is expected only if validation is being aborted due to externalizations ahead.
		// cancellation on this condition is triggered in validateCue.
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= proto.BlockHeight(slot))
		reply <- scpkit.MaybeValidValue
		return
	}
	must.NoError(ctx, err)
	must.Assert(ctx, cvx != nil)

	if protoutil.HashCVFast(cvx) == cvc.Cv_hash {
		reply <- scpkit.FullyValidValue
	} else {
		reply <- scpkit.InvalidValue
	}
}
