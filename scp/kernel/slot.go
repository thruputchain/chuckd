package kernel

import proto "github.com/groundhogchain/protocol-go"

type TimerID int

const (
	NominationTimer TimerID = iota
	BallotProtocolTimer
)

type Slot struct{}

func (slot *Slot) getSCP() *SCP {
	panic("XXX")
}

func (slot *Slot) Driver() Driver {
	panic("XXX")
}

func (slot *Slot) recordStatement(stmt *proto.SCPStatement) {
	panic("XXX")
}

func (slot *Slot) getSlotIndex() proto.SCPSlotIndex {
	panic("XXX")
}

func (slot *Slot) setFullyValidated(s bool) {
	panic("XXX")
}

func (slot *Slot) getQuorumSetFromStatement(stmt *proto.SCPStatement) *proto.SCPQuorumSet {
	panic("XXX")
}

func (slot *Slot) getLatestCompositeCandidate() proto.SCPValue {
	panic("XXX")
}

func (slot *Slot) getLocalNode() *LocalNode {
	panic("XXX")
}

func (slot *Slot) createEnvelope(stmt *proto.SCPStatement) *proto.SCPEnvelope {
	panic("XXX")
}

func (slot *Slot) processEnvelope(env *proto.SCPEnvelope, self bool) EnvelopeState {
	panic("XXX")
}

func (slot *Slot) isFullyValidated() bool {
	panic("XXX")
}
