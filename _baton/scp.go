package _baton

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
	// XXX: check valid (stacks on top of the existing ones)
	// update state
	extProp := extProp{Slot: slotIndex, Proposal: &prop}
	x.baton.scp.lk.Lock()
	defer x.baton.scp.lk.Unlock()
	x.baton.scp.prop = append(x.baton.scp.prop, &proposal{ext: extProp})
	// notify baton
	x.baton.extProp <- extProp
}

type extProp struct {
	Slot     api.Slot
	Proposal *proto.Proposal
}
