package kernel

import proto "github.com/groundhogchain/protocol-go"

type Slot struct{}

func (slot *Slot) recordStatement(stmt *proto.SCPStatement) {
	panic("XXX")
}

func (slot *Slot) getSlotIndex() proto.SCPSlotIndex {
	panic("XXX")
}
