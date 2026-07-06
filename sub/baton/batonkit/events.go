package batonkit

import (
	"fmt"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type EventNominating struct {
	Nominating any
	Onto       proto.DbTip
	NumTx      int
	Height     proto.BlockHeight
	ValueHash  proto.ConsensusValueHash
	//
	ConsensusBasis *proto.ConsensusBasis
	ConsensusValue *proto.ConsensusValue
}

func (x EventNominating) String() string {
	return fmt.Sprintf("NOMINATING: height %d; tx_count %d; value_hash %s; scp_value: %s",
		x.Height, x.NumTx, protoutil.ValueDisplay(x.ConsensusValue),
		protoutil.SCPValueDisplay(x.ConsensusValue),
	)
}

type EventNominated struct {
	Nominated any
	Onto      proto.DbTip
	NumTx     int
	Height    proto.BlockHeight
	ValueHash proto.ConsensusValueHash
	//
	ConsensusBasis *proto.ConsensusBasis
	ConsensusValue *proto.ConsensusValue
}

func (x EventNominated) String() string {
	return fmt.Sprintf("NOMINATED: height %d; tx_count %d; value %s; value_hash %s",
		x.Height, x.NumTx,
		protoutil.ValueDisplay(x.ConsensusValue),
		protoutil.ValueDisplay(x.ConsensusValue))
}

type EventCommitted struct {
	Committed any
	Onto      proto.DbTip
	Height    proto.BlockHeight
	ValueHash proto.ConsensusValueHash
	//
	DbgValidatable *CommittableBlock
}

func (x EventCommitted) String() string {
	value := x.DbgValidatable.ConsensusValue()
	return fmt.Sprintf("COMMITTED: height %d; tx_count %d; value %s; value_hash %s; proposer %s",
		x.Height, x.DbgValidatable.NumTransactions(),
		protoutil.ValueDisplay(value),
		protoutil.ValueDisplay(value),
		protoutil.VIDString(x.DbgValidatable.AsConsensusBasis().Value.Proposer))
}

type EventValidating struct {
	Validating any
	Onto       proto.DbTip
	NumTx      int
	Height     proto.BlockHeight
	ValueHash  proto.ConsensusValueHash
	//
	ConsensusBasis *proto.ConsensusBasis
	ConsensusValue *proto.ConsensusValue
}

func (x EventValidating) String() string {
	return fmt.Sprintf("VALIDATING: height %d; tx_count %d; value %s; value_hash %s; proposer %s",
		x.Height, x.NumTx,
		protoutil.ValueDisplay(x.ConsensusValue),
		protoutil.ValueDisplay(x.ConsensusValue),
		protoutil.VIDString(x.ConsensusBasis.Value.Proposer))
}

type EventValidated struct {
	Validated any
	Onto      proto.DbTip
	NumTx     int
	Height    proto.BlockHeight
	ValueHash proto.ConsensusValueHash
	//
	ConsensusBasis *proto.ConsensusBasis
	ConsensusValue *proto.ConsensusValue
}

func (x EventValidated) String() string {
	return fmt.Sprintf("VALIDATED: height %d; tx_count %d; value %s; value_hash %s; proposer %s",
		x.Height, x.NumTx,
		protoutil.ValueDisplay(x.ConsensusValue),
		protoutil.ValueDisplay(x.ConsensusValue),
		protoutil.VIDString(x.ConsensusBasis.Value.Proposer))
}
