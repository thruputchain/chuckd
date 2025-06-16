package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/validate"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/protoutil"
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

	nominations := make(protoutil.NominationsByHash, 0, len(values))
	for _, v := range values {
		var p proto.Nomination
		if err := xdrutil.UnmarshalXdr(&p, v); err != nil {
			liblog.Errorf("unable to decode nomination for combining: %v", err)
			break
		}
		nominations = append(nominations, &p)
	}
	nominations.Sort()
	if len(nominations) == 0 {
		return nil
	} else {
		return x.baton.nomM.MarshalXdr(nominations[0])
	}
}

func (x scpHandler) ValidateValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value, nomination bool) scpapi.ValidationLevel {
	<-x.baton.scp

	liblog.Infof("baton validate value for slot %d received", slot)
	defer liblog.Infof("baton validate value completed")

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
	return <-reply
}

func (b *Baton) validateValue(ctx context.Context, nomination *proto.Nomination, reply chan<- scpapi.ValidationLevel) {

	b.lock()
	dbTip := b.tip
	numExt := len(b.externalized)
	b.unlock()

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

	// this context will be cancelled upon new externalizations or commits
	vCtx, vCtxCancel := context.WithCancel(ctx)
	defer vCtxCancel()
	go func() {
		b.WaitGrowBeyond(vCtx, nomination.Block_height-1) // cancel ctx if tip or ext moves
		vCtxCancel()
	}()

	txs, ok := b.handler.WaitTxs(ctx, nomination.Transaction_hashes)
	if !ok {
		liblog.Infof("cannot validate, missing txs")
		reply <- scpapi.MaybeValidValue
		return
	}

	ptxs := make([]*proto.ProposedTransaction, len(txs))
	for i, tx := range txs {
		ptxs[i] = &proto.ProposedTransaction{
			Signed_tx:                      *tx,
			Nondeterministic_proposal_data: nomination.Transaction_nominations[i].Nondeterministic_proposal_data,
		}
	}

	vb := &batonapi.ValidatableBlock{
		BlockHeight:          nomination.Block_height,
		BlockHash:            nomination.Block_hash,
		PrevBlockHash:        nomination.Prev_block_hash,
		ProposedTransactions: ptxs,
		DbTip:                dbTip,
	}

	go func() {
		blockID, err := validate.ValidateClose(ctx, b.handler, vb, b.callsPerExec, b.batchSize)
		if err == batonapi.ErrNotValid {
			liblog.Infof("cannot validate, inconsistent")
			reply <- scpapi.InvalidValue
			return
		}
		if err != nil {
			liblog.Infof("cannot validate, other: %v", err)
			reply <- scpapi.MaybeValidValue
			return
		}
		reply <- scpapi.FullyValidValue

		b.lock()
		defer b.unlock()

		// cache validated blocks
		b.validated[nomination.Block_hash] = validated{Nomination: nomination, BlockID: blockID}
	}()
}

func (x scpHandler) ExtractValidValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) *scpapi.Value {
	<-x.baton.scp
	return nil
}

func (x scpHandler) ExternalizedValue(ctx context.Context, slot scpapi.Slot, value scpapi.Value) {
	<-x.baton.scp

	var nomination proto.Nomination
	if err := xdrutil.UnmarshalXdr(&nomination, value); err != nil {
		return
	}

	// check that SCP slot equals nomination's block height
	if slot != nomination.Block_height {
		liblog.Errorf("scp slot index does not equal proposed block height")
		return
	}

	x.baton.lock()
	defer x.baton.unlock()

	// check validity and update state
	ok := isValidExternalization(&nomination, x.baton.externalized)
	if ok {
		x.baton.externalized = append(x.baton.externalized, &nomination)
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
