package scpkit

import (
	"context"
	"time"

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
	SCPEmitEnvelope(context.Context, proto.SCPEnvelope) error
}

type SCPGetQuorumSet interface {
	SCPGetQuorumSetLocally(context.Context, proto.SCPHash) (*proto.SCPQuorumSet, error)
}

type SCPHandler interface {
	SCPCombineCandidates(context.Context, Slot, []Value) Value
	SCPValidateNominationValue(ctx context.Context, slot Slot, value Value) ValidationLevel
	SCPValidateBallotValue(ctx context.Context, slot Slot, value Value) ValidationLevel
	SCPExtractValidValue(ctx context.Context, slot Slot, value Value) *Value
	SCPExternalizedValue(ctx context.Context, slot Slot, value Value)
}

type SCPController interface {
	SCPNominate(ctx context.Context, slot Slot, value Value, prev Value) bool
	SCPStopNomination(ctx context.Context, slot Slot)
	SCPUpdateQuorumSet(ctx context.Context, qset proto.SCPQuorumSet)
	SCPReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope)
	SCPKnownQSets
}

type SCPKnownQSets interface {
	SCPSetKnownQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) proto.SCPOptQuorumSet
	SCPGetKnownQuorumSet(ctx context.Context, qhash proto.SCPHash) proto.SCPOptQuorumSet
	SCPRemoveKnownQuorumSet(ctx context.Context, qhash proto.SCPHash) proto.SCPOptQuorumSet
}

type SCPCommander interface {
	SCPBackfeedEnvelopes(ctx context.Context, since Slot, timeout time.Duration)
	SCPInitLowestNonExternalized(ctx context.Context, slot Slot)
	SCPReviveStateFromEnvelopes(ctx context.Context, slot Slot)
	// connects scp to envelopes from other validators
	SCPAcceptReceiveEnvelopes(ctx context.Context)
}

type SCPBroker interface {
	SCPCommander
	SCPController
}
