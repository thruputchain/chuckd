package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/baton/validate"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) SCPHandler() api.SCPHandler {
	return scpHandler{b}
}

type scpHandler struct {
	baton *Baton
}

func (x scpHandler) CombineCandidates(ctx context.Context, slot api.Slot, values []api.Value) api.Value {
	<-x.baton.scp

	proposals := make(protoutil.NominationsByHash, 0, len(values))
	for _, v := range values {
		var p proto.Nomination
		if err := xdrutil.UnmarshalXdr(&p, v); err != nil {
			liblog.Errorf("unable to decode proposal for combining: %v", err)
			break
		}
		proposals = append(proposals, &p)
	}
	proposals.Sort()
	if len(proposals) == 0 {
		return nil
	} else {
		return xdrutil.MarshalXdr(proposals[0])
	}
}

func (x scpHandler) ValidateValue(ctx context.Context, slot api.Slot, value api.Value, nomination bool) api.ValidationLevel {
	<-x.baton.scp

	liblog.Infof("baton validate value for slot %d received", slot)
	defer liblog.Infof("baton validate value completed")

	var prop proto.Nomination
	err := xdrutil.UnmarshalXdr(&prop, value)
	if err != nil {
		return api.InvalidValue
	}
	if len(prop.Transaction_hashes) != len(prop.Transaction_nominations) {
		return api.InvalidValue
	}

	reply := make(chan api.ValidationLevel, 1)
	x.baton.validateValue(ctx, &prop, reply)
	return <-reply
}

func (b *Baton) validateValue(ctx context.Context, proposal *proto.Nomination, reply chan<- api.ValidationLevel) {

	b.lock(ctx)
	dbTip := b.tip
	numExt := len(b.externalized)
	b.unlock(ctx)

	if proposal.Block_height <= dbTip.Height {
		liblog.Infof("cannot validate, proposal below tip")
		reply <- api.InvalidValue
		return
	}
	if proposal.Block_height > dbTip.Height+1 {
		liblog.Infof("cannot validate, proposal far ahead")
		reply <- api.MaybeValidValue
		return
	}
	if proposal.Prev_block_hash != dbTip.Hash {
		liblog.Infof("cannot validate, previous block %v != db tip %v",
			form.Display(proposal.Prev_block_hash), form.Display(dbTip.Hash))
		reply <- api.InvalidValue
		return
	}
	if numExt > 0 {
		liblog.Infof("cannot validate, newer externalizations not processed")
		reply <- api.InvalidValue // there are newer externalizations
		return
	}

	// this context will be cancelled upon new externalizations or commits
	vCtx, vCtxCancel := context.WithCancel(ctx)
	defer vCtxCancel()
	go func() {
		b.WaitGrowBeyond(vCtx, proposal.Block_height-1) // cancel ctx if tip or ext moves
		vCtxCancel()
	}()

	txs, ok := b.handler.WaitTxs(ctx, proposal.Transaction_hashes)
	if !ok {
		liblog.Infof("cannot validate, missing txs")
		reply <- api.MaybeValidValue
		return
	}

	ptxs := make([]*proto.ProposedTransaction, len(txs))
	for i, tx := range txs {
		ptxs[i] = &proto.ProposedTransaction{
			Signed_tx:                      *tx,
			Nondeterministic_proposal_data: proposal.Transaction_nominations[i].Nondeterministic_proposal_data,
		}
	}

	vb := &sdk.ValidatableBlock{
		BlockHeight:          proposal.Block_height,
		BlockHash:            proposal.Block_hash,
		PrevBlockHash:        proposal.Prev_block_hash,
		ProposedTransactions: ptxs,
		DbTip:                dbTip,
	}

	go func() {
		blockID, err := validate.ValidateClose(ctx, b.handler, vb)
		if err == sdk.ErrNotValid {
			liblog.Infof("cannot validate, inconsistent")
			reply <- api.InvalidValue
			return
		}
		if err != nil {
			liblog.Infof("cannot validate, other: %v", err)
			reply <- api.MaybeValidValue
			return
		}
		reply <- api.FullyValidValue

		b.lock(ctx)
		defer b.unlock(ctx)

		// cache validated blocks
		b.validated[proposal.Block_hash] = validated{Nomination: proposal, BlockID: blockID}
	}()
}

func (x scpHandler) ExtractValidValue(ctx context.Context, slot api.Slot, value api.Value) *api.Value {
	<-x.baton.scp
	return nil
}

func (x scpHandler) ExternalizedValue(ctx context.Context, slot api.Slot, value api.Value) {
	<-x.baton.scp

	var proposal proto.Nomination
	if err := xdrutil.UnmarshalXdr(&proposal, value); err != nil {
		return
	}

	// check that SCP slot equals proposal's block height
	if slot != proposal.Block_height {
		liblog.Errorf("scp slot index does not equal proposed block height")
		return
	}

	x.baton.lock(ctx)
	defer x.baton.unlock(ctx)

	// check validity and update state
	ok := isValidExternalization(&proposal, x.baton.externalized)
	if ok {
		x.baton.externalized = append(x.baton.externalized, &proposal)
	}
}

func isValidExternalization(p *proto.Nomination, ps []*proto.Nomination) bool {
	if len(ps) == 0 {
		return true
	}
	last := ps[len(ps)-1]
	if p.Block_height <= last.Block_height {
		return false
	}
	if p.Block_height == last.Block_height+1 && p.Prev_block_hash != last.Block_hash {
		return false
	}
	return true
}
