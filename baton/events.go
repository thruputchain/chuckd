package baton

import proto "github.com/groundhogchain/protocol-go"

type EventNominating struct {
	Nominating any
	Tip        Tip
	NumTx      int
	Proposal   *proto.Proposal
}

type EventNominated struct {
	Nominated any
	Tip       Tip
	NumTx     int
	Proposal  *proto.Proposal
}

type EventCommitted struct {
	Committed any
	*ValidatableBlock
	proto.BlockID
}

type EventValidated struct {
	Validated any
	*ValidatableBlock
	proto.BlockID
}
