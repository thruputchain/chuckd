package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/log"
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
	proposals := make(protoutil.ProposalsByHash, 0, len(values))
	for _, v := range values {
		var p proto.Proposal
		if err := xdrutil.UnmarshalXdr(&p, v); err != nil {
			log.Errorf("unable to decode proposal for combining: %v", err)
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

	var prop proto.Proposal
	err := xdrutil.UnmarshalXdr(&prop, value)
	if err != nil {
		return api.InvalidValue
	}

	x.baton.lock()
	dbTip := x.baton.db.tip
	pendingExternalizations := !x.baton.consensus.ext.IsEmpty()
	x.baton.unlock()

	if prop.Block_height <= dbTip.Height {
		return api.InvalidValue
	}
	if prop.Block_height > dbTip.Height+1 {
		return api.MaybeValidValue
	}
	if prop.Prev_block_hash != dbTip.Hash {
		return api.InvalidValue
	}
	if pendingExternalizations {
		return api.InvalidValue // there are newer externalizations
	}

	ptxs := []*proto.ProposedTransaction{}
	for _, b := range prop.Proposed_transaction_batches {
		btxs, ok := x.baton.handler.LookupBatch(ctx, b)
		if !ok {
			return api.MaybeValidValue
		}
		ptxs = append(ptxs, btxs...)
	}

	vb := &ValidatableBlock{
		BlockHeight:          prop.Block_height,
		BlockHash:            prop.Block_hash,
		PrevBlockHash:        prop.Prev_block_hash,
		ProposedTransactions: ptxs,
		DbTip:                dbTip,
	}

	blockID, err := x.baton.validateClose(ctx, vb)

	// cache validated blocks
	x.baton.consensus.lk.Lock()
	x.baton.consensus.validatedProposal[prop.Block_hash] = &prop
	x.baton.consensus.validatedBlockID[prop.Block_hash] = &blockID
	x.baton.consensus.lk.Unlock()

	return api.FullyValidValue
}

func (x scpHandler) ExtractValidValue(ctx context.Context, slot api.Slot, value api.Value) *api.Value {
	return nil
}

func (x scpHandler) ExternalizedValue(ctx context.Context, slot api.Slot, value api.Value) {
	var prop proto.Proposal
	if err := xdrutil.UnmarshalXdr(&prop, value); err != nil {
		return
	}

	// check that SCP slot equals proposal's block height
	if slot != prop.Block_height {
		log.Errorf("scp slot index does not equal proposed block height")
		return
	}

	// check validity and update state
	x.baton.consensus.lk.Lock()

	externalized := proposal{slot: slot, proto: &prop}
	ok := isValidExternalization(externalized, x.baton.consensus.ext)
	if ok {
		x.baton.consensus.ext = append(x.baton.consensus.ext, externalized)
	}

	x.baton.consensus.lk.Unlock()

	// notify baton
	if ok {
		x.baton.externalized <- externalized
	}
}

func isValidExternalization(prop proposal, ps proposals) bool {
	if len(ps) == 0 {
		return true
	}
	last := ps[len(ps)-1]
	if prop.BlockHeight() <= last.BlockHeight() {
		return false
	}
	if prop.BlockHeight() == last.BlockHeight()+1 && prop.PrevBlockHash() != last.BlockHash() {
		return false
	}
	return true
}
