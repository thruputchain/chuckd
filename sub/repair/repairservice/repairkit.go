package repairservice

import proto "github.com/thruputchain/thruput-protocol-go"

type RepairProvider interface {
	// Announce the cv of our current nomination.
	// This information is used to verify missing txs requests are for our nominations only.
	AnnounceNomination(value *proto.ConsensusValue)
}
