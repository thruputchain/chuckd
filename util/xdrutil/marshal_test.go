package xdrutil

import (
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestMarshal(t *testing.T) {

	m := NewXdrMarshaler()

	x := proto.SoloNomination{Slot_index: 1}
	var y proto.SoloNomination
	err := UnmarshalXdr(&y, m.MarshalXdr(&x).Bytes())
	if err != nil {
		t.Error(err)
	}
	if x.Slot_index != y.Slot_index {
		t.Errorf("expecting equality")
	}
}
