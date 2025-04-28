package baton

import (
	"github.com/groundhogchain/chuckd/scp/api"
)

func (x *Baton) CombineCandidates(api.Slot, []api.Value) api.Value {
	panic("XXX")
}

func (x *Baton) ValidateValue(slotIndex api.Slot, value api.Value, nomination bool) api.ValidationLevel {
	panic("XXX")
}

func (x *Baton) ExtractValidValue(slotIndex api.Slot, value api.Value) *api.Value {
	panic("XXX")
}

func (x *Baton) OnExternalizedValue(slotIndex api.Slot, value api.Value) {
	panic("XXX")
}
