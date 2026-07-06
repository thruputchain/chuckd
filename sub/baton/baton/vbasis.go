package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) validateBasis(
	ctx context.Context,
	slot scpkit.Slot,
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) {
	header := &value.Proposal_header

	liblog.Infof(ctx, "validating cv: height %d; value %s; proposer %s",
		slot, protoutil.ValueDisplay(value), protoutil.VIDString(value.Proposer))

	if b.tryValidateBasisFast(ctx, slot, basis, value, reply) {
		return
	}

	var dbTip *proto.DbTip
	var extTip *externalizedBasis

	dbTip, extTip, err := b.WaitCommittedAt(ctx, header.Block_height-1)
	if err != nil {
		switch {
		case errors.Is(err, ErrTipAlreadyAhead):
			must.Assert(ctx, b.HighestCommitted() >= header.Block_height)
			b.validateCommittedValue(ctx, slot, basis, value, reply)
			return
		case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
			highestExternalized := b.HighestCommittedOrExternalized()
			liblog.Infof(ctx, "skipping validation due to externalizations ahead: height: %d; highest_externalized: %d",
				header.Block_height, highestExternalized)
			must.Assert(ctx, highestExternalized >= header.Block_height)
			reply <- scpkit.MaybeValidValue
			return
		}
		panic("unreachable")
	}

	tipHeight := protoutil.TipHeight(*dbTip)
	must.Assert(ctx, tipHeight+1 == header.Block_height)

	extValueHash := protoutil.HashValueToProto(extTip.ConsensusValue())
	if value.Prev_value_hash != extValueHash {
		liblog.Errorf(ctx, "cannot validate, previous value hash mismatch, nomination %v vs ours %v",
			libxdr.HashString(value.Prev_value_hash), libxdr.HashString(extValueHash))
		reply <- scpkit.InvalidValue
		return
	}

	if header.Block_height != tipHeight+1 {
		liblog.Errorf(ctx, "cannot validate, block height mismatch, got %d want %d",
			header.Block_height, tipHeight+1)
		reply <- scpkit.InvalidValue
		return
	}

	if header.Network_id != extTip.ConsensusValue().Proposal_header.Network_id {
		liblog.Errorf(ctx, "cannot validate, network id mismatch")
		reply <- scpkit.InvalidValue
		return
	}

	prevStamp := protoutil.TimeFromTimepoint(extTip.ConsensusValue().Proposal_header.Timestamp)
	newStamp := protoutil.TimeFromTimepoint(header.Timestamp)
	if !prevStamp.Before(newStamp) {
		liblog.Errorf(ctx, "cannot validate, timestamp not increasing")
		reply <- scpkit.InvalidValue
		return
	}
	now := b.handler.Now()
	if !newStamp.Before(now) {
		liblog.Errorf(ctx, "cannot validate, timestamp in the future (stamp %s, now %s)",
			libtime.FmtTime(newStamp), libtime.FmtTime(now))
		reply <- scpkit.InvalidValue
		return
	}

	b.lk.Lock()
	alreadyInvalidated := b.isInvalidated(value)
	b.lk.Unlock()
	if alreadyInvalidated {
		reply <- scpkit.InvalidValue
		return
	}

	validatedTip, err := b.validateProposal(ctx, *dbTip, basis, value)
	if errors.Is(err, batonkit.ErrNotValid) {
		b.lk.Lock()
		b.setInvalidated(basis, value)
		b.lk.Unlock()
		reply <- scpkit.InvalidValue
		return
	}
	if err != nil {
		must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		must.Assert(ctx, b.HighestCommittedOrExternalized() >= header.Block_height)
		reply <- scpkit.MaybeValidValue
		return
	}

	liblog.Debugf(ctx, "caching validation for block %d and consensus %s",
		header.Block_height, protoutil.ValueDisplay(value))
	b.lk.Lock()
	b.setValidated(basis, value, validatedTip.Id)
	b.lk.Unlock()

	reply <- scpkit.FullyValidValue
}

func (b *Baton) tryValidateBasisFast(
	ctx context.Context,
	slot scpkit.Slot,
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
	reply chan<- scpkit.ValidationLevel,
) bool {
	header := &value.Proposal_header

	if header.Block_height != proto.BlockHeight(slot) {
		reply <- scpkit.InvalidValue
		return true
	}
	if xdrutil.TokenSizeOfBatch(basis.Proposal_txs.Hash) !=
		xdrutil.TokenSizeOfBatch(basis.Proposal_txs.Nondeterministic_data) {
		reply <- scpkit.InvalidValue
		return true
	}

	b.lk.Lock()
	dbTipOnRequest := b.tipDb
	alreadyValidated := b.getValidatedBlockID(value)
	b.lk.Unlock()

	if !protoutil.IsSaneBasis(basis) {
		liblog.Infof(ctx, "cannot validate, proposal txs not sane")
		reply <- scpkit.InvalidValue
		return true
	}

	if header.Block_height <= protoutil.TipHeight(dbTipOnRequest) {
		b.validateCommittedValue(ctx, slot, basis, value, reply)
		return true
	}

	if alreadyValidated != nil {
		reply <- scpkit.FullyValidValue
		return true
	}

	return false
}
