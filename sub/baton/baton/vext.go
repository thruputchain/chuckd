package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
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
	liblog.Debugf("validating cue for already externalized slot %d", slot)

	cv := b.handler.FindConsensusValue(ctx, proto.BlockHeight(slot))
	must.Assert(ctx, cv != nil)

	if b.ConsensusValueAgreesWithCue(cv, cvc) {
		reply <- scpkit.FullyValidValue
	} else {
		reply <- scpkit.InvalidValue
	}
}

func (b *Baton) validateExternalizedConsensus(
	ctx context.Context,
	slot scpkit.Slot,
	cv *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) {
	liblog.Debugf("validating consensus value for already externalized slot %d", slot)

	cvx := b.handler.FindConsensusValue(ctx, proto.BlockHeight(slot))
	must.Assert(ctx, cvx != nil)

	if b.hashCV(cvx) == b.hashCV(cv) {
		reply <- scpkit.FullyValidValue
	} else {
		reply <- scpkit.InvalidValue
	}
}

func (b *Baton) ConsensusValueAgreesWithCue(cv *proto.ConsensusValue, cvc *proto.ConsensusValueCue) bool {
	return b.hashCV(cv) == cvc.Cv_hash &&
		cvc.Block_hash == cv.Nomination.Block_hash &&
		cvc.Prev_block_hash == cv.Nomination.Prev_block_hash &&
		cvc.Prev_cv_hash == cv.Prev_cv_hash
}
