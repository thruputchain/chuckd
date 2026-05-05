package baton

import (
	"context"

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

	cv := b.handler.CVPoolFindCommitted(ctx, proto.BlockHeight(slot))
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

	cvx := b.handler.CVPoolFindCommitted(ctx, proto.BlockHeight(slot))
	must.Assert(ctx, cvx != nil)

	if b.hashCV(cvx) == cvc.Cv_hash {
		reply <- scpkit.FullyValidValue
	} else {
		reply <- scpkit.InvalidValue
	}
}
