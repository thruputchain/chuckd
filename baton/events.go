package baton

import proto "github.com/groundhogchain/protocol-go"

type EventNominating struct {
	Nominating any
	Tip        Tip
	NumTx      int
	Proposal   *proto.Proposal
}
