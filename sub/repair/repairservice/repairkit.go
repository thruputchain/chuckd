package repairservice

import proto "github.com/groundhogchain/protocol-go"

type RepairProvider interface {
	// Announce the cv hash of our current nomination.
	// This information is used to verify missing txs requests are for our nominations only.
	AnnounceNomination(cue *proto.ConsensusValueCue)
}
