package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) SCPHandler() scpkit.SCPHandler {
	return scpHandler{b}
}

type scpHandler struct {
	baton *Baton
}

func (x scpHandler) CombineCandidates(ctx context.Context, slot scpkit.Slot, values []scpkit.Value) scpkit.Value {
	<-x.baton.ready

	if len(values) == 0 {
		return nil
	} else {
		scpkit.Values(values).Sort()
		return values[0]
	}
}

func (x scpHandler) ValidateBallotValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) (level scpkit.ValidationLevel) {
	return x.ValidateCue(ctx, slot, value, false)
}

func (x scpHandler) ValidateNominationValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) (level scpkit.ValidationLevel) {
	return x.ValidateCue(ctx, slot, value, true)
}

func (x scpHandler) ExtractValidValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) *scpkit.Value {
	<-x.baton.ready
	return &value
}

func (x scpHandler) ExternalizedValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) {
	<-x.baton.ready

	var cvc proto.ConsensusValueCue
	if err := xdrutil.UnmarshalXdr(&cvc, value); err != nil {
		liblog.Errorf("scp externalized value for slot %d which cannot be decoded", slot)
		return
	}

	// if we participated in this round, the cv shold be in our cv pool,
	// otherwise it is too late to repair a missing cv value.
	cv := x.baton.handler.LookupConsensusValue(ctx, cvc.Cv_hash)

	x.baton.lock()
	defer x.baton.unlock()

	if cv == nil {

		extCue := externalizedCue{height: proto.BlockHeight(slot), cvc: &cvc}
		x.baton.placeExternalization(ctx, extCue)

	} else {

		// check that SCP slot equals nomination's block height
		if slot != proto.SCPSlotIndex(cv.Nomination.Block_height) {
			liblog.Errorf("scp slot index does not equal proposed block height")
			return
		}

		extConsensus := x.baton.newExternalized(cv)
		if x.baton.placeExternalization(ctx, extConsensus) {
			x.baton.handler.PersistConsensusValue(ctx, *cv)
		}

	}
}

// placeExternalization returns true if ext has not been placed previously, and is not at or below db tip.
// scp can produce duplicate externalizations, which will result in placeExternalization returning false.
func (b *Baton) placeExternalization(ctx context.Context, ext externalized) bool {
	if ext.Height() <= b.tipDb.Height {
		return false
	}
	if len(b.externalized) == 0 { // no prior externalizations
		b.externalized = append(b.externalized, ext)
		return true
	}
	lastExt := b.externalized[len(b.externalized)-1]
	if ext.Height() <= lastExt.Height() {
		// if the externalization is within the range of already seen externalizations,
		// only place it if it upgrades an externalized cue to an externalized consensus value
		if extConsensus, ok := ext.(externalizedConsensus); ok {
			for i, priorExt := range b.externalized {
				if priorExt.Height() == extConsensus.Height() {
					if _, priorIsCue := priorExt.(externalizedCue); priorIsCue {
						must.Assert(ctx, extConsensus.BlockHash() == priorExt.BlockHash())
						must.Assert(ctx, extConsensus.PrevBlockHash() == priorExt.PrevBlockHash())
						b.externalized[i] = extConsensus
						return true
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
		must.Assertf(ctx,
			ext.Height() == lastExt.Height()+1 && ext.PrevBlockHash() == lastExt.BlockHash(),
			"scp externalization is invalid",
		)
		b.externalized = append(b.externalized, ext)
		return true
	}
}
