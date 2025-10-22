package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
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
	return x.ValidateValue(ctx, slot, value, false)
}

func (x scpHandler) ValidateNominationValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) (level scpkit.ValidationLevel) {
	return x.ValidateValue(ctx, slot, value, true)
}

func (x scpHandler) ValidateValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value, nomination bool) (level scpkit.ValidationLevel) {
	<-x.baton.ready

	kind := "ballot"
	if nomination {
		kind = "nomination"
	}
	liblog.Infof("baton validate %s value for slot %d received", kind, slot)
	defer func() {
		liblog.Infof("baton validate %s value for slot %d completed with %v", kind, slot, scpkit.ValidationLevelString(level))
	}()

	var cv proto.ConsensusValue
	err := xdrutil.UnmarshalXdr(&cv, value)
	if err != nil {
		return scpkit.InvalidValue
	}
	if len(cv.Nomination.Transaction_hashes) != len(cv.Nomination.Transaction_nominations) {
		return scpkit.InvalidValue
	}

	reply := make(chan scpkit.ValidationLevel, 1)
	x.baton.validateValue(ctx, &cv, reply)
	level = <-reply
	return
}

func (b *Baton) validateValue(ctx context.Context, cv *proto.ConsensusValue, reply chan<- scpkit.ValidationLevel) {

	b.lock()
	dbTipOnRequest := b.tipDb
	alreadyValidated := b.getValidated(cv)
	b.unlock()

	if cv.Nomination.Block_height <= dbTipOnRequest.Height {
		liblog.Infof("cannot validate, nomination below tip")
		reply <- scpkit.InvalidValue
		return
	}

	if alreadyValidated != nil {
		reply <- scpkit.FullyValidValue
		return
	}

	go func() {

		// Step 1: cancel validation request if externalization arise

		// this context will be cancelled upon new externalizations or commits
		vCtx, vCtxCancel := context.WithCancel(ctx)
		go func() {
			// waiter is cancelled whenever validation completes or externalization occurs
			b.WaitExternalizedBeyond(vCtx, cv.Nomination.Block_height) // cancel ctx if tip or ext moves
			vCtxCancel()
		}()
		defer vCtxCancel()

		// Step 2: wait for the block below the validation request to be reached and committed

		var dbTip *proto.DbTip
		var extTip *externalized

		var err error
		dbTip, extTip, err = b.WaitCommittedAt(vCtx, cv.Nomination.Block_height-1)
		if err != nil { // context cancelled
			if errors.Is(err, ErrTipAlreadyAhead) {
				reply <- scpkit.InvalidValue
				return
			}
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			reply <- scpkit.InvalidValue
			return
		}

		must.Assert(ctx, dbTip.Height+1 == cv.Nomination.Block_height)

		// Step 3: sanity check nomination

		if cv.Nomination.Prev_block_hash != dbTip.Hash {
			liblog.Infof("cannot validate, previous block mismatch, nomination %v vs ours %v",
				form.Display(cv.Nomination.Prev_block_hash),
				form.Display(dbTip.Hash))
			reply <- scpkit.InvalidValue
			return
		}

		if cv.Prev_cv_hash != extTip.ConsensusValueHash() {
			liblog.Infof("cannot validate, previous consensus hash mismatch, nomination %v vs ours %v",
				form.Display(cv.Prev_cv_hash),
				form.Display(extTip.ConsensusValueHash()))
			reply <- scpkit.InvalidValue
			return
		}

		// check timestamp

		prevStamp := protoutil.TimeFromTimepoint(extTip.ConsensusValue().Timestamp)
		newStamp := protoutil.TimeFromTimepoint(cv.Timestamp)
		if !prevStamp.Before(newStamp) {
			liblog.Infof("cannot validate, timestamp not increasing")
			reply <- scpkit.InvalidValue
			return
		}
		if !newStamp.Before(b.handler.Now()) {
			liblog.Infof("cannot validate, timestamp in the future")
			reply <- scpkit.InvalidValue
			return
		}

		// Step 4: start validation

		validatedTip, err := b.validate(vCtx, *dbTip, cv)
		if err != nil {
			// err can only ever be context cancelled or deadline exceeded.
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			// it occurs only if scp has externalized newer blocks and thus
			// the validation request is not relevant any longer.
			// in this case, returning invalid value does not violate scp honest semantics.
			reply <- scpkit.InvalidValue
			return
		}

		// cache validated blocks
		b.lock()
		liblog.Debugf("caching validation for %s", form.Display(cv.Nomination.Block_hash))
		b.setValidated(cv, validatedTip.Id)
		b.unlock()

		reply <- scpkit.FullyValidValue
	}()
}

func (x scpHandler) ExtractValidValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) *scpkit.Value {
	<-x.baton.ready
	return &value
}

func (x scpHandler) ExternalizedValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value) {
	<-x.baton.ready

	var cv proto.ConsensusValue
	if err := xdrutil.UnmarshalXdr(&cv, value); err != nil {
		return
	}

	// check that SCP slot equals nomination's block height
	if slot != proto.SCPSlotIndex(cv.Nomination.Block_height) {
		liblog.Errorf("scp slot index does not equal proposed block height")
		return
	}

	x.baton.lock()
	defer x.baton.unlock()

	ext := x.baton.newExternalized(&cv)
	if isNewExternalization(ctx, x.baton.tipDb, ext, x.baton.externalized) {
		x.baton.handler.PersistConsensusValue(ctx, cv)
		x.baton.externalized = append(x.baton.externalized, ext)
	}
}

// isNewExternalization checks that an externalization has not already been seen;
// scp can produce duplicate externalizations.
func isNewExternalization(
	ctx context.Context,
	tip proto.DbTip,
	ext externalized,
	priorExts []externalized,
) bool {
	if ext.Height() <= tip.Height {
		return false
	}
	if len(priorExts) == 0 {
		return true // no prior externalizations
	}
	lastExt := priorExts[len(priorExts)-1]
	if ext.Height() <= lastExt.Height() {
		return false
	}
	must.Assertf(ctx,
		ext.Height() != lastExt.Height()+1 || ext.PrevBlockHash() == lastExt.BlockHash(),
		"scp externalization is invalid",
	)
	return true
}
