package api

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type Slot = proto.SoloSlotIndex
type NodeID = proto.ValidatorID
type Value = proto.SoloValue

type ValidationLevel = proto.SoloValidationLevel

const (
	InvalidValue    = proto.SoloInvalidValue
	MaybeValidValue = proto.SoloMaybeValidValue
	FullyValidValue = proto.SoloFullyValidValue
)

// when to accept a nomination? validated by majority?

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
	ValidateValue(ctx context.Context, slot Slot, value Value, nomination bool) ValidationLevel

	// ExtractValidValue transforms the value, if possible to a different
	// value that the local node would agree to (fully validated).
	// This is used during nomination when encountering an invalid value (ie
	// validateValue did not return `FullyValidatedValue` for this value).
	// returning nil means no valid value could be extracted.
	ExtractValidValue(ctx context.Context, slot Slot, value Value) *Value

	// OnExternalizedValue is is called at most once per slot when the slot
	// externalizes its value
	OnExternalizedValue(ctx context.Context, slot Slot, value Value)
}

type SCPController interface {
	// Submit a value to consider for slot
	// prev is the value from slot-1.
	// prev is used to pick a leader.
	// Nomination is blocking and can be aborted using the context.
	Nominate(ctx context.Context, slot Slot, value Value, prev Value) bool

	// Local QuorumSet interface (can be dynamically updated)
	UpdateQuorumSet(proto.QuorumSet)
	GetQuorumSet() proto.QuorumSet
	NodeID() NodeID

	// Purges all data relative to all the slots whose slot is smaller
	// than the specified `maxSlotIndex` except for slotToKeep slot.
	PurgeSlots(maxSlotIndex Slot, slotToKeep Slot)

	// Returns whether the local node is a validator.
	IsValidator() bool // not just listening

	// Returns the validation state of the given slot
	IsSlotFullyValidated() bool
}
