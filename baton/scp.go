package baton

import (
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (x *Baton) CombineCandidates(slotIndex api.Slot, values []api.Value) api.Value {
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

func (x *Baton) ValidateValue(slotIndex api.Slot, value api.Value, nomination bool) api.ValidationLevel {
	panic("XXX")
}

func (x *Baton) ExtractValidValue(slotIndex api.Slot, value api.Value) *api.Value {
	return nil
}

func (x *Baton) OnExternalizedValue(slotIndex api.Slot, value api.Value) {
	panic("XXX")
}
