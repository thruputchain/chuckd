package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
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

// During nomination, if a value is considered invalid (by ValidateNominationValue), scp calls ExtractValidValue to
// give the application a chance to propose a derivative value that is valid. Stellar uses this call to strip invalid
// upgrade proposals from an otherwise valid value.
// If ExtractValidValue is not used, it must return nil!
// Returning non-nil asserts that the returned value is valid.
// In particular, returning the value passed to ExtractValidValue is a security vulnerability, because
// scp calls ExtractValidValue only after ValidateNominationValue returned false on that value, and so
// returning that value is essentially asserting validity on a value that is definitely not valid.
func (x scpHandler) ExtractValidValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) *scpkit.Value {
	<-x.baton.ready
	return nil
}

func (x scpHandler) ExternalizedValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) {
	<-x.baton.ready

	var cvc proto.ConsensusValueCue
	if err := libxdr.UnmarshalXdr(&cvc, value); err != nil {
		liblog.Fatalf("scp externalized value for slot %d which cannot be decoded", slot)
		return
	}

	liblog.Infof("received externalization for slot %d with block hash %s and consensus value hash %s",
		slot,
		libxdr.HashString(cvc.Block_hash),
		libxdr.HashString(cvc.Cv_hash))

	// if we participated in this round, the cv shold be in our cv pool,
	// otherwise it is too late to repair a missing cv value.
	cv := x.baton.handler.ConsensusValuePoolLookup(ctx, cvc.Cv_hash)

	x.baton.lock()
	defer x.baton.unlock()

	x.baton.noteExternalized(slot)

	if cv == nil {

		extCue := externalizedCue{height: proto.BlockHeight(slot), cvc: &cvc}
		x.baton.placeExternalization(ctx, extCue)

	} else {

		must.Assert(ctx, slot == proto.SCPSlotIndex(cv.Nomination.Block_height))

		extConsensus := x.baton.newExternalized(cv, cvc.Cv_hash)
		if x.baton.placeExternalization(ctx, extConsensus) {
			// XXX: Is CV persistence necessary here, considering it also happens in commit?
			t0 := time.Now()
			x.baton.handler.PersistConsensusValue(ctx, *cv)
			persistCVDelay := time.Since(t0)
			x.baton.reportPersistCV(cv.Nomination.Block_height, persistCVDelay)
			liblog.Debugf("cv_persist_delay=%v | height=%d", persistCVDelay, cv.Nomination.Block_height)
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
