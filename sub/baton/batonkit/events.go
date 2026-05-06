package batonkit

import (
	"fmt"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type EventNominating struct {
	Nominating any
	Onto       proto.DbTip
	NumTx      int
	Height     proto.BlockHeight
	BlockHash  proto.BlockHash
	//
	DbgConsensusValue    *proto.ConsensusValue
	DbgConsensusValueCue *proto.ConsensusValueCue
}

func (x EventNominating) String() string {
	return fmt.Sprintf("NOMINATING: height %d; tx_count %d; block_hash %s; scp_value: %s",
		x.Height, x.NumTx, libxdr.HashString(x.BlockHash),
		protoutil.SCPValueDisplay(x.DbgConsensusValueCue),
	)
}

type EventNominated struct {
	Nominated any
	Onto      proto.DbTip
	NumTx     int
	Height    proto.BlockHeight
	BlockHash proto.BlockHash
	//
	DbgConsensusValue    *proto.ConsensusValue
	DbgConsensusValueCue *proto.ConsensusValueCue
}

func (x EventNominated) String() string {
	return fmt.Sprintf("NOMINATED: height %d; tx_count %d; cvc %s; block_hash %s",
		x.Height, x.NumTx,
		protoutil.CVCDisplay(x.DbgConsensusValueCue),
		libxdr.HashString(x.BlockHash))
}

type EventCommitted struct {
	Committed any
	Onto      proto.DbTip
	Height    proto.BlockHeight
	BlockHash proto.BlockHash
	//
	DbgValidatable *CommittableBlock
}

func (x EventCommitted) String() string {
	cue := x.DbgValidatable.ConsensusValueCue()
	return fmt.Sprintf("COMMITTED: height %d; tx_count %d; cvc %s; block_hash %s; proposer %s",
		x.Height, x.DbgValidatable.NumTransactions(),
		protoutil.CVCDisplay(cue),
		libxdr.HashString(x.BlockHash),
		protoutil.VPublicShortString(x.DbgValidatable.AsConsensusValue().Nomination.Proposer))
}

type EventValidating struct {
	Validating any
	Onto       proto.DbTip
	NumTx      int
	Height     proto.BlockHeight
	BlockHash  proto.BlockHash
	//
	DbgConsensusValue    *proto.ConsensusValue
	DbgConsensusValueCue *proto.ConsensusValueCue
}

func (x EventValidating) String() string {
	return fmt.Sprintf("VALIDATING: height %d; tx_count %d; cvc %s; block_hash %s; proposer %s",
		x.Height, x.NumTx,
		protoutil.CVCDisplay(x.DbgConsensusValueCue),
		libxdr.HashString(x.BlockHash),
		protoutil.VPublicShortString(x.DbgConsensusValue.Nomination.Proposer))
}

type EventValidated struct {
	Validated any
	Onto      proto.DbTip
	NumTx     int
	Height    proto.BlockHeight
	BlockHash proto.BlockHash
	//
	DbgConsensusValue    *proto.ConsensusValue
	DbgConsensusValueCue *proto.ConsensusValueCue
}

func (x EventValidated) String() string {
	return fmt.Sprintf("VALIDATED: height %d; tx_count %d; cvc %s; block_hash %s; proposer %s",
		x.Height, x.NumTx,
		protoutil.CVCDisplay(x.DbgConsensusValueCue),
		libxdr.HashString(x.BlockHash),
		protoutil.VPublicShortString(x.DbgConsensusValue.Nomination.Proposer))
}
