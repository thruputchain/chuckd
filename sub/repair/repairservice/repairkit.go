package repairservice

import proto "github.com/groundhogchain/protocol-go"

type RepairProvider interface {
	// Sets the cv hash of our current nomination.
	// This information is used to verify missing txs requests are for our nominations only.
	SetCurrentNomination(cvh *proto.ConsensusValueHash)
}
