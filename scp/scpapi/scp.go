package scpapi

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type Slot = proto.SCPSlotIndex
type NodeID = proto.ValidatorID
type Value = proto.SCPValue

type ValidationLevel = proto.SCPValidationLevel

func ToSoloValidationLevel(vl ValidationLevel) proto.SoloValidationLevel {
	switch vl {
	case proto.SCP_FULLY_VALID:
		return proto.SoloFullyValidValue
	case proto.SCP_MAYBE_VALID:
		return proto.SoloMaybeValidValue
	case proto.SCP_INVALID:
		return proto.SoloInvalidValue
	}
	panic("bug")
}

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
	GetQuorumSet(context.Context, proto.SCPHash) (*proto.SCPQuorumSet, error)
}

type SCPHandler interface {

	// combine multiple candidates into a composite value
	CombineCandidates(context.Context, Slot, []Value) Value

	// ValidateValue is called on each message received before any processing
	// is done. It should be used to filter out values that are not compatible
	// with the current state of that node. Invalid values can never
	// externalize.
	// If the value cannot be validated (node is missing some context) but
	// passes
	// the validity checks, MaybeValidValue can be returned. This will cause
	// the current slot to be marked as a non validating slot: the local node
	// will abstain from emitting its position.
	// validation can be *more* restrictive during nomination as needed
	// NB: validation levels are ordered
	ValidateNominationValue(ctx context.Context, slot Slot, value Value) ValidationLevel
	ValidateBallotValue(ctx context.Context, slot Slot, value Value) ValidationLevel

	// ExtractValidValue transforms the value, if possible to a different
	// value that the local node would agree to (fully validated).
	// This is used during nomination when encountering an invalid value (ie
	// validateValue did not return `FullyValidatedValue` for this value).
	// returning nil means no valid value could be extracted.
	ExtractValidValue(ctx context.Context, slot Slot, value Value) *Value

	// ExternalizedValue is is called at most once per slot when the slot
	// externalizes its value
	ExternalizedValue(ctx context.Context, slot Slot, value Value)
}

type SCPController interface {
	// Submit a value to consider for slot
	// prev is the value from slot-1.
	// prev is used to pick a leader.
	// Nomination blocks until the round is over, and can be aborted using the context.
	Nominate(ctx context.Context, slot Slot, value Value, prev Value) bool

	// Local QuorumSet interface (can be dynamically updated)
	UpdateQuorumSet(proto.QuorumSet)
}

type SCPReceivedEnvelope interface {
	ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope)
}
