package xdrutil

import (
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestMarshal(t *testing.T) {

	x := proto.SoloNomination{Slot_index: 1}
	var y proto.SoloNomination
	err := UnmarshalXdr(&y, MarshalXdr(&x))
	if err != nil {
		t.Error(err)
	}
	if x.Slot_index != y.Slot_index {
		t.Errorf("expecting equality")
	}
}
