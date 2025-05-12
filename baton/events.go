package baton

import proto "github.com/groundhogchain/protocol-go"

type EventNominating struct {
	Nominating any
	TipHeight  proto.BlockHeight
	TipHash    proto.BlockHash
	TipID      proto.BlockID
	NumTx      int
}
