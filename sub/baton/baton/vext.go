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

func (b *Baton) validateExternalizedValue(
	ctx context.Context,
	slot scpkit.Slot,
	value *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) {
	liblog.Debugf(ctx, "validating value for already externalized slot %d", slot)

	basis, err := b.handler.CVPoolFindCommitted(ctx, proto.BlockHeight(slot))
	if errors.Is(err, context.Canceled) {
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= proto.BlockHeight(slot))
		reply <- scpkit.MaybeValidValue
		return
	}
	must.NoError(ctx, err)
	must.Assert(ctx, basis != nil)

	if protoutil.BasisAgreesWithValue(basis, value) {
		reply <- scpkit.FullyValidValue
	} else {
		reply <- scpkit.InvalidValue
	}
}

func (b *Baton) validateCommittedValue(
	ctx context.Context,
	slot scpkit.Slot,
	_ *proto.ConsensusBasis,
	value *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) {
	liblog.Debugf(ctx, "validating consensus value for already externalized slot %d", slot)

	basisX, err := b.handler.CVPoolFindCommitted(ctx, proto.BlockHeight(slot))
	if errors.Is(err, context.Canceled) {
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= proto.BlockHeight(slot))
		reply <- scpkit.MaybeValidValue
		return
	}
	must.NoError(ctx, err)
	must.Assert(ctx, basisX != nil)

	if protoutil.BasisAgreesWithValue(basisX, value) {
		reply <- scpkit.FullyValidValue
	} else {
		reply <- scpkit.InvalidValue
	}
}
