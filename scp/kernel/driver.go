package kernel

import (
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

// `validateValue` is called on each message received before any processing
// is done. It should be used to filter out values that are not compatible
// with the current state of that node. Invalid values can never
// externalize.
// If the value cannot be validated (node is missing some context) but
// passes
// the validity checks, kMaybeValidValue can be returned. This will cause
// the current slot to be marked as a non validating slot: the local node
// will abstain from emitting its position.
// validation can be *more* restrictive during nomination as needed
// NB: validation levels are ordered
type ValidationLevel int

const (
	InvalidValue        ValidationLevel = 0 // value is invalid for sure
	MaybeValidValue                     = 1 // value may be valid
	FullyValidatedValue                 = 2 // value is valid for sure
)

type Driver interface {
	StartedBallotProtocol(slotIndex proto.SCPSlotIndex, ballot proto.SCPBallot)

	ComputeTimeout(counter proto.SCPCount) time.Duration
	StartTimer(slotIndex proto.SCPSlotIndex, timerID TimerID, timeout time.Duration, cb func())
	StopTimer(slotIndex proto.SCPSlotIndex, timerID TimerID)
}
