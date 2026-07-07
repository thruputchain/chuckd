package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) SCPHandler() scpkit.SCPHandler {
	return scpHandler{b}
}

type scpHandler struct {
	baton *Baton
}

func (x scpHandler) SCPCombineCandidates(ctx context.Context, slot scpkit.Slot, values []scpkit.Value) scpkit.Value {
	ctx = liblog.Enterf(ctx, "SCPHandler.SCPCombineCandidates", "slot=%d", slot)
	defer func() { liblog.Leavef(ctx, "slot=%d", slot) }()

	<-x.baton.ready

	if len(values) == 0 {
		return nil
	} else {
		scpkit.Values(values).Sort()
		return values[0]
	}
}

func (x scpHandler) SCPValidateBallotValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) (level scpkit.ValidationLevel) {
	ctx = liblog.Enterf(ctx, "SCPHandler.SCPValidateBallotValue", "slot=%d", slot)
	defer func() { liblog.Leavef(ctx, "slot=%d; validation=%v", slot, level) }()

	return x.ValidateValue(ctx, slot, value, false)
}

func (x scpHandler) SCPValidateNominationValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) (level scpkit.ValidationLevel) {
	ctx = liblog.Enterf(ctx, "SCPHandler.SCPValidateNominationValue", "slot=%d", slot)
	defer func() { liblog.Leavef(ctx, "slot=%d; validation=%v", slot, level) }()

	return x.ValidateValue(ctx, slot, value, true)
}

// During nomination, if a value is considered invalid (by ValidateNominationValue), scp calls SCPExtractValidValue to
// give the application a chance to propose a derivative value that is valid. Stellar uses this call to strip invalid
// upgrade proposals from an otherwise valid value.
// If SCPExtractValidValue is not used, it must return nil!
// Returning non-nil asserts that the returned value is valid.
// In particular, returning the value passed to SCPExtractValidValue is a security vulnerability, because
// scp calls SCPExtractValidValue only after ValidateNominationValue returned false on that value, and so
// returning that value is essentially asserting validity on a value that is definitely not valid.
func (x scpHandler) SCPExtractValidValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) *scpkit.Value {
	ctx = liblog.Enterf(ctx, "SCPHandler.SCPExtractValidValue", "slot=%d", slot)
	defer func() { liblog.Leavef(ctx, "slot=%d", slot) }()

	<-x.baton.ready
	return nil
}

func (x scpHandler) SCPExternalizedValue(ctx context.Context, slot scpkit.Slot, extValue scpkit.Value) {
	ctx = liblog.Enterf(ctx, "SCPHandler.SCPExternalizedValue", "slot=%d", slot)
	defer func() { liblog.Leavef(ctx, "slot=%d", slot) }()

	<-x.baton.ready

	var value proto.ConsensusValue
	if err := libxdr.UnmarshalXdrEOF(&value, extValue); err != nil {
		liblog.Fatalf(ctx, "scp externalized value for slot %d which cannot be decoded", slot)
		return
	}

	liblog.Infof(ctx, "received externalization for slot %d with value hash %s",
		slot,
		protoutil.ValueDisplay(&value))

	liblog.Debugf(ctx, "notified scp broker of lowest non-externalized slot %d", slot+1)

	// if we participated in this round, the basis shold be in our basis pool,
	// otherwise it is too late to repair a missing basis.
	basis, err := x.baton.handler.CVPoolLookup(ctx, value)
	must.NoError(ctx, err)

	x.baton.lock()
	defer x.baton.unlock()

	x.baton.noteExternalized(slot)

	if basis == nil {

		extValue := externalizedValue{height: proto.BlockHeight(slot), value: &value}
		x.baton.placeExternalization(ctx, extValue)

	} else {

		must.Assert(ctx, slot == proto.SCPSlotIndex(protoutil.ValueHeight(&basis.Value)))

		extConsensus := newExternalized(&value, basis)
		x.baton.placeExternalization(ctx, extConsensus)
	}
}

// placeExternalization returns true if ext has not been placed previously, and is not at or below db tip.
// scp can produce duplicate externalizations, which will result in placeExternalization returning false.
func (b *Baton) placeExternalization(ctx context.Context, ext externalized) bool {
	if ext.Height() <= protoutil.ValueHeight(&b.tipDb.Value) {
		return false
	}
	if len(b.externalized) == 0 { // no prior externalizations
		b.externalized = append(b.externalized, ext)
		return true
	}
	lastExt := b.externalized[len(b.externalized)-1]
	if ext.Height() <= lastExt.Height() {
		// if the externalization is within the range of already seen externalizations,
		// only place it if it upgrades an externalized value to an externalized basis
		if extConsensus, ok := ext.(externalizedBasis); ok {
			for i, priorExt := range b.externalized {
				if priorExt.Height() == extConsensus.Height() {
					if _, priorIsValue := priorExt.(externalizedValue); priorIsValue {
						if externalizedAgree(extConsensus, priorExt) {
							b.externalized[i] = extConsensus
							return true
						} else {
							liblog.Fatalf(ctx, "externalization mismatch: new %v vs prior %v", extConsensus, priorExt)
						}
					} else {
						return false
					}
				}
			}
			panic("unreachable")
		} else {
			return false
		}
	} else {
		if ext.Height() == lastExt.Height()+1 {
			must.Assert(ctx, externalizedStack(lastExt, ext))
		}
		b.externalized = append(b.externalized, ext)
		return true
	}
}
