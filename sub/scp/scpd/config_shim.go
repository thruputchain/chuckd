package scpd

import (
	proto "github.com/thruputchain/thruput-protocol-go"
)

type SCPConfig_shim struct {
	Max_timeout_ms                  uint64            `json:"max_timeout_ms"`
	Initial_ballot_timeout_ms       uint64            `json:"initial_ballot_timeout_ms"`
	Increment_ballot_timeout_ms     uint64            `json:"increment_ballot_timeout_ms"`
	Initial_nomination_timeout_ms   uint64            `json:"initial_nomination_timeout_ms"`
	Increment_nomination_timeout_ms uint64            `json:"increment_nomination_timeout_ms"`
	Quorum_set                      SCPQuorumSet_shim `json:"quorum_set"`
}

type SCPQuorumSet_shim struct {
	Threshold  uint32              `json:"threshold"`
	Validators []SCPNodeID_shim    `json:"validators"`
	Inner_sets []SCPQuorumSet_shim `json:"inner_sets"`
}

type SCPNodeID_shim [32]byte

func SCPConfigToShim(scp proto.SCPConfig) SCPConfig_shim {
	return SCPConfig_shim{
		Max_timeout_ms:                  scp.Max_timeout_ms,
		Initial_ballot_timeout_ms:       scp.Initial_ballot_timeout_ms,
		Increment_ballot_timeout_ms:     scp.Increment_ballot_timeout_ms,
		Initial_nomination_timeout_ms:   scp.Initial_nomination_timeout_ms,
		Increment_nomination_timeout_ms: scp.Increment_nomination_timeout_ms,
		Quorum_set:                      SCPQuorumSetToShim(scp.Quorum_set),
	}
}

func SCPQuorumSetToShim(qs proto.SCPQuorumSet) SCPQuorumSet_shim {
	validators := make([]SCPNodeID_shim, len(qs.Validators))
	for i, validator := range qs.Validators {
		validators[i] = SCPNodeIDToShim(validator)
	}

	innerSets := make([]SCPQuorumSet_shim, len(qs.Inner_sets))
	for i, innerSet := range qs.Inner_sets {
		innerSets[i] = SCPQuorumSetToShim(innerSet)
	}

	return SCPQuorumSet_shim{
		Threshold:  uint32(qs.Threshold),
		Validators: validators,
		Inner_sets: innerSets,
	}
}

func SCPNodeIDToShim(nodeID proto.SCPNodeID) SCPNodeID_shim {
	return SCPNodeID_shim(*nodeID.Ed25519())
}
