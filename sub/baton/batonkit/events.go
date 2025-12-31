package batonkit

import (
	"fmt"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type EventNominating struct {
	Nominating any
	Onto       proto.DbTip
	NumTx      int
	Height     proto.BlockHeight
	Hash       proto.BlockHash
	//
	DbgConsensusValue    *proto.ConsensusValue
	DbgConsensusValueCue *proto.ConsensusValueCue
}

func (x EventNominating) String() string {
	return fmt.Sprintf("NOMINATING: height %d; tx_count %d; hash %s", x.Height, x.NumTx, libxdr.HashString(x.Hash))
}

type EventNominated struct {
	Nominated any
	Onto      proto.DbTip
	NumTx     int
	Height    proto.BlockHeight
	Hash      proto.BlockHash
	//
	DbgConsensusValue    *proto.ConsensusValue
	DbgConsensusValueCue *proto.ConsensusValueCue
}

func (x EventNominated) String() string {
	return fmt.Sprintf("NOMINATED: height %d; tx_count %d; hash %s", x.Height, x.NumTx, libxdr.HashString(x.Hash))
}

type EventCommitted struct {
	Committed any
	Onto      proto.DbTip
	Height    proto.BlockHeight
	Hash      proto.BlockHash
	//
	DbgValidatable *CommittableBlock
}

func (x EventCommitted) String() string {
	return fmt.Sprintf("COMMITTED: height %d; tx_count %d; hash %s", x.Height, x.DbgValidatable.NumTransactions(), libxdr.HashString(x.Hash))
}

type EventValidating struct {
	Validating any
	Onto       proto.DbTip
	NumTx      int
	Height     proto.BlockHeight
	Hash       proto.BlockHash
}

func (x EventValidating) String() string {
	return fmt.Sprintf("VALIDATING: height %d; tx_count %d; hash %s", x.Height, x.NumTx, libxdr.HashString(x.Hash))
}

type EventValidated struct {
	Validated any
	Onto      proto.DbTip
	NumTx     int
	Height    proto.BlockHeight
	Hash      proto.BlockHash
}

func (x EventValidated) String() string {
	return fmt.Sprintf("VALIDATED: height %d; tx_count %d; hash %s", x.Height, x.NumTx, libxdr.HashString(x.Hash))
}
