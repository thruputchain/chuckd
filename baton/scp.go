package baton

import (
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

func (x scpHandler) CombineCandidates(slotIndex api.Slot, values []api.Value) api.Value {
	proposals := make(protoutil.ProposalsBySize, 0, len(values))
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

func (x scpHandler) ValidateValue(slotIndex api.Slot, value api.Value, nomination bool) api.ValidationLevel {
	panic("XXX")
}

func (x scpHandler) ExtractValidValue(slotIndex api.Slot, value api.Value) *api.Value {
	return nil
}

func (x scpHandler) OnExternalizedValue(slotIndex api.Slot, value api.Value) {
	var prop proto.Proposal
	if err := xdrutil.UnmarshalXdr(&prop, value); err != nil {
		return
	}

	// XXX: check integrity (signatures?)

	// check that SCP slot equals proposal's block height
	if slotIndex != prop.Block_height {
		log.Errorf("scp slot index does not equal proposed block height")
		return
	}

	// XXX: check valid (stacks on top of the existing ones)

	// update state
	x.baton.scp.lk.Lock()
	defer x.baton.scp.lk.Unlock()

	externalized := proposal{slot: slotIndex, proto: &prop}

	x.baton.scp.prop = append(x.baton.scp.prop, externalized)

	// notify baton
	x.baton.externalized <- externalized
}
