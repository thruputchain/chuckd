package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) SCPHandler() scpapi.SCPHandler {
	return scpHandler{b}
}

type scpHandler struct {
	baton *Baton
}

func (x scpHandler) CombineCandidates(ctx context.Context, slot scpapi.Slot, values []scpapi.Value) scpapi.Value {
	<-x.baton.scp

	if len(values) == 0 {
		return nil
	} else {
		scpapi.Values(values).Sort()
		return values[0]
	}

	// nominations := make(protoutil.NominationsByHash, 0, len(values))
	// for _, v := range values {
	// 	var p proto.Nomination
	// 	if err := xdrutil.UnmarshalXdr(&p, v); err != nil {
	// 		liblog.Errorf("unable to decode nomination for combining: %v", err)
	// 		break
	// 	}
	// 	nominations = append(nominations, &p)
	// }
	// nominations.Sort()
	// if len(nominations) == 0 {
	// 	return nil
	// } else {
	// 	return x.baton.nomM.MarshalXdr(nominations[0]).Bytes()
	// }
}

func (x scpHandler) ValidateBallotValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) (level scpapi.ValidationLevel) {
	return x.ValidateValue(ctx, slot, value, false)
}

func (x scpHandler) ValidateNominationValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) (level scpapi.ValidationLevel) {
	return x.ValidateValue(ctx, slot, value, true)
}

func (x scpHandler) ValidateValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value, nomination bool) (level scpapi.ValidationLevel) {
	<-x.baton.scp

	kind := "ballot"
	if nomination {
		kind = "nomination"
	}
	liblog.Infof("baton validate %s value for slot %d received", kind, slot)
	defer func() {
		liblog.Infof("baton validate %s value for slot %d completed with %v", kind, slot, scpapi.ValidationLevelString(level))
	}()

	var prop proto.Nomination
	err := xdrutil.UnmarshalXdr(&prop, value)
	if err != nil {
		return scpapi.InvalidValue
	}
	if len(prop.Transaction_hashes) != len(prop.Transaction_nominations) {
		return scpapi.InvalidValue
	}

	reply := make(chan scpapi.ValidationLevel, 1)
	x.baton.validateValue(ctx, &prop, reply)
	level = <-reply
	return
}

func (b *Baton) validateValue(ctx context.Context, nomination *proto.Nomination, reply chan<- scpapi.ValidationLevel) {

	b.lock()
	dbTip := b.tip
	numExt := len(b.externalized)
	_, alreadyValidated := b.validated[nomination.Block_hash]
	b.unlock()

	if alreadyValidated {
		reply <- scpapi.FullyValidValue
		return
	}

	if nomination.Block_height <= dbTip.Height {
		liblog.Infof("cannot validate, nomination below tip")
		reply <- scpapi.InvalidValue
		return
	}
	if nomination.Block_height > dbTip.Height+1 {
		liblog.Infof("cannot validate, nomination far ahead")
		//XXX: wait until we reach height
		reply <- scpapi.MaybeValidValue
		return
	}
	if nomination.Prev_block_hash != dbTip.Hash {
		liblog.Infof("cannot validate, previous block %v != db tip %v",
			form.Display(nomination.Prev_block_hash), form.Display(dbTip.Hash))
		reply <- scpapi.InvalidValue
		return
	}
	if numExt > 0 {
		liblog.Infof("cannot validate, newer externalizations not processed")
		reply <- scpapi.InvalidValue // there are newer externalizations
		return
	}

	go func() {
		// this context will be cancelled upon new externalizations or commits
		vCtx, vCtxCancel := context.WithCancel(ctx)
		go func() {
			// waiter is cancelled whenever validation completes or externalization occurs
			b.WaitGrowBeyond(vCtx, nomination.Block_height-1) // cancel ctx if tip or ext moves
			vCtxCancel()
		}()
		defer vCtxCancel()

		validatedTip, err := b.validate(vCtx, dbTip, nomination)
		if err != nil {
			// err can only ever be context cancelled or deadline exceeded.
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			// it occurs only if scp has externalized newer blocks and thus
			// the validation request is not relevant any longer.
			// in this case, returning invalid value does not violate scp honest semantics.
			reply <- scpapi.InvalidValue
			return
		}

		// cache validated blocks
		b.lock()
		liblog.Debugf("caching validation for %s", form.Display(nomination.Block_hash))
		b.validated[nomination.Block_hash] = validated{
			Nomination: nomination,
			BlockID:    validatedTip.Id,
		}
		b.unlock()

		reply <- scpapi.FullyValidValue
	}()
}

func (x scpHandler) ExtractValidValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) *scpapi.Value {
	<-x.baton.scp
	return &value
}

func (x scpHandler) ExternalizedValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) {
	<-x.baton.scp

	var nomination proto.Nomination
	if err := xdrutil.UnmarshalXdr(&nomination, value); err != nil {
		return
	}

	// check that SCP slot equals nomination's block height
	if slot != proto.SCPSlotIndex(nomination.Block_height) {
		liblog.Errorf("scp slot index does not equal proposed block height")
		return
	}

	x.baton.lock()
	defer x.baton.unlock()

	checkExternalizationCompatible(ctx, x.baton.tip, &nomination, x.baton.externalized)
	x.baton.externalized = append(x.baton.externalized, &nomination)
}

// checkExternalizationCompatible checks that an externalization is compatible with the state of our system
func checkExternalizationCompatible(ctx context.Context, tip proto.DbTip, ext *proto.Nomination, priorExts []*proto.Nomination) {
	must.Assertf(ctx, ext.Block_height > tip.Height,
		"externalization height %d is not above db %d", ext.Block_height, tip.Height)
	if len(priorExts) == 0 {
		return // no prior externalizations
	}
	last := priorExts[len(priorExts)-1]
	must.Assert(ctx, ext.Block_height > last.Block_height)
	must.Assert(ctx, ext.Block_height != last.Block_height+1 || ext.Prev_block_hash == last.Block_hash)
}
