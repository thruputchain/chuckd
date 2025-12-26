package scpkit

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type Slot = proto.SCPSlotIndex
type NodeID = proto.SCPNodeID
type Value = proto.SCPValue

type ValidationLevel = proto.SCPValidationLevel

func ValidationLevelString(vl ValidationLevel) string {
	switch vl {
	case proto.SCP_FULLY_VALID:
		return "FullyValid"
	case proto.SCP_MAYBE_VALID:
		return "MaybeValid"
	case proto.SCP_INVALID:
		return "Invalid"
	}
	panic("bug")
}

const (
	InvalidValue    = proto.SCP_INVALID
	MaybeValidValue = proto.SCP_MAYBE_VALID
	FullyValidValue = proto.SCP_FULLY_VALID
)

type SCPEmitEnvelope interface {
	EmitEnvelope(context.Context, proto.SCPEnvelope) error
}

type SCPGetQuorumSet interface {
	GetQuorumSetLocally(context.Context, proto.SCPHash) (*proto.SCPQuorumSet, error)
}

type SCPHandler interface {
	CombineCandidates(context.Context, Slot, []Value) Value
	ValidateNominationValue(ctx context.Context, slot Slot, value Value) ValidationLevel
	ValidateBallotValue(ctx context.Context, slot Slot, value Value) ValidationLevel
	ExtractValidValue(ctx context.Context, slot Slot, value Value) *Value
	ExternalizedValue(ctx context.Context, slot Slot, value Value)
}

type SCPController interface {
	Nominate(ctx context.Context, slot Slot, value Value, prev Value) bool
	StopNomination(ctx context.Context, slot Slot)
	UpdateQuorumSet(ctx context.Context, qset proto.SCPQuorumSet)
	ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope)
	SCPKnownQSets
}

type SCPKnownQSets interface {
	SetKnownQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) proto.SCPOptQuorumSet
	GetKnownQuorumSet(ctx context.Context, qhash proto.SCPHash) proto.SCPOptQuorumSet
	RemoveKnownQuorumSet(ctx context.Context, qhash proto.SCPHash) proto.SCPOptQuorumSet
}
