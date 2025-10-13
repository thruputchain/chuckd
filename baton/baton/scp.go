package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
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
			// XXX: cancel the waiter of validation completes
			b.WaitGrowBeyond(vCtx, nomination.Block_height-1) // cancel ctx if tip or ext moves
			vCtxCancel()
		}()
		defer vCtxCancel()

		validatedTip, err := b.validate(vCtx, dbTip.Proto(), nomination)
		if err != nil {
			// err can only ever be context cancelled or deadline exceeded.
			// it occurs only if scp has externalized newer blocks and thus
			// the validation request is not relevant any longer.
			// therefore, returning invalid value is not in violation of scp honest semantics.
			reply <- scpapi.InvalidValue
			return
		}

		// cache validated blocks
		b.lock()
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

	// check validity and update state
	ok := isValidExternalization(x.baton.tip, &nomination, x.baton.externalized)
	if ok {
		x.baton.externalized = append(x.baton.externalized, &nomination)
	}
}

func isValidExternalization(tip batonapi.Tip, p *proto.Nomination, ps []*proto.Nomination) bool {
	if len(ps) == 0 {
		return true
	}
	last := ps[len(ps)-1]
	if p.Block_height <= tip.Height {
		return false
	}
	if p.Block_height <= last.Block_height {
		return false
	}
	if p.Block_height == last.Block_height+1 && p.Prev_block_hash != last.Block_hash {
		return false
	}
	return true
}
