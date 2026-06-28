package baton

import (
	"fmt"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	proto "github.com/groundhogchain/protocol-go"
)

type ViewBlock interface {
	Proposer() proto.ValidatorID
	BlockHeight() proto.BlockHeight
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
	BasisHash() proto.ConsensusBasisHash
	PrevBasisHash() proto.ConsensusBasisHash
	Timestamp() proto.TimePoint
	Upgrades() []proto.Upgrade
	PrevValue() proto.ConsensusValueHash
	//
	IsMissing() bool
	IsExtBasis() bool
	IsExtValue() bool
	IsCatchup() bool
	IsDb() bool
	//
	Committable() batonkit.Committable
	Icon() string
}

func BlockValue(b ViewBlock) *proto.ConsensusValue {
	return &proto.ConsensusValue{
		Timestamp:       b.Timestamp(),
		Proposer:        b.Proposer(),
		Upgrades:        b.Upgrades(),
		Block_height:    b.BlockHeight(),
		Prev_value_hash: b.PrevValue(),
		Basis_hash:      b.BasisHash(),
		Prev_basis_hash: b.PrevBasisHash(),
		Block_hash:      b.BlockHash(),
		Prev_block_hash: b.PrevBlockHash(),
	}
}

// db block

type ViewDbBlock struct {
	TipDb  proto.DbTip
	TipExt externalizedBasis
}

func (x ViewDbBlock) Proposer() proto.ValidatorID    { return x.TipExt.Proposer() }
func (x ViewDbBlock) BlockHash() proto.BlockHash     { return x.TipDb.Block_hash }
func (x ViewDbBlock) BlockHeight() proto.BlockHeight { return x.TipDb.Height }
func (x ViewDbBlock) PrevBlockHash() proto.BlockHash { panic("should not be called") }
func (x ViewDbBlock) BasisHash() proto.ConsensusBasisHash {
	return x.TipExt.ConsensusBasisHash()
}
func (x ViewDbBlock) PrevBasisHash() proto.ConsensusBasisHash {
	return x.TipExt.ConsensusBasis().Prev_basis_hash
}
func (x ViewDbBlock) Timestamp() proto.TimePoint { return x.TipExt.ConsensusValue().Timestamp }
func (x ViewDbBlock) Upgrades() []proto.Upgrade  { return x.TipExt.ConsensusValue().Upgrades }
func (x ViewDbBlock) PrevValue() proto.ConsensusValueHash {
	return x.TipExt.ConsensusValue().Prev_value_hash
}
func (x ViewDbBlock) IsMissing() bool                   { return false }
func (x ViewDbBlock) IsExtBasis() bool                  { return false }
func (x ViewDbBlock) IsExtValue() bool                  { return false }
func (x ViewDbBlock) IsCatchup() bool                   { return false }
func (x ViewDbBlock) IsDb() bool                        { return true }
func (x ViewDbBlock) Committable() batonkit.Committable { panic("should not be called") }
func (x ViewDbBlock) Icon() string                      { return fmt.Sprintf("D:%d", x.TipDb.Height) }

// catchup block

type ViewCatchupBlock struct {
	Linked    bool // all catchup blocks between db and first scp chain hashes correctly
	Candidate CatchupArrival
	Validated *proto.BlockID
}

func (x ViewCatchupBlock) Proposer() proto.ValidatorID    { return x.Candidate.Proposer() }
func (x ViewCatchupBlock) BlockHash() proto.BlockHash     { return x.Candidate.BlockHash() }
func (x ViewCatchupBlock) BlockHeight() proto.BlockHeight { return x.Candidate.BlockHeight() }
func (x ViewCatchupBlock) PrevBlockHash() proto.BlockHash { return x.Candidate.PrevBlockHash() }
func (x ViewCatchupBlock) BasisHash() proto.ConsensusBasisHash {
	return x.Candidate.ConsensusBasisHash()
}
func (x ViewCatchupBlock) PrevBasisHash() proto.ConsensusBasisHash {
	return x.Candidate.ConsensusBasis().Prev_basis_hash
}
func (x ViewCatchupBlock) Timestamp() proto.TimePoint { return x.Candidate.ConsensusValue().Timestamp }
func (x ViewCatchupBlock) Upgrades() []proto.Upgrade  { return x.Candidate.ConsensusValue().Upgrades }
func (x ViewCatchupBlock) PrevValue() proto.ConsensusValueHash {
	return x.Candidate.ConsensusValue().Prev_value_hash
}
func (x ViewCatchupBlock) IsMissing() bool  { return false }
func (x ViewCatchupBlock) IsExtBasis() bool { return false }
func (x ViewCatchupBlock) IsExtValue() bool { return false }
func (x ViewCatchupBlock) IsCatchup() bool  { return true }
func (x ViewCatchupBlock) IsDb() bool       { return false }
func (x ViewCatchupBlock) Committable() batonkit.Committable {
	if !x.Linked {
		return nil
	}
	return batonkit.NewCommittableCatchup(
		x.Candidate.Block(),
		x.Candidate.ConsensusValue(),
		x.Candidate.SignedTransactions(),
		x.Validated,
	)
}
func (x ViewCatchupBlock) Icon() string {
	if x.Linked {
		return fmt.Sprintf("L:%d", x.Candidate.BlockHeight())
	} else {
		return fmt.Sprintf("C:%d", x.Candidate.BlockHeight())
	}
}

// externalized consensus block

type ViewExtBasisBlock struct {
	Ext       externalizedBasis
	Validated *proto.BlockID
	Catchup   *CatchupArrival
}

func (x ViewExtBasisBlock) Proposer() proto.ValidatorID { return x.Ext.Proposer() }
func (x ViewExtBasisBlock) BlockHash() proto.BlockHash {
	return x.Ext.BlockHash()
}
func (x ViewExtBasisBlock) BlockHeight() proto.BlockHeight {
	return x.Ext.Height()
}
func (x ViewExtBasisBlock) PrevBlockHash() proto.BlockHash {
	return x.Ext.PrevBlockHash()
}
func (x ViewExtBasisBlock) BasisHash() proto.ConsensusBasisHash {
	return x.Ext.ConsensusBasisHash()
}
func (x ViewExtBasisBlock) PrevBasisHash() proto.ConsensusBasisHash {
	return x.Ext.PrevConsensusBasisHash()
}
func (x ViewExtBasisBlock) Timestamp() proto.TimePoint { return x.Ext.ConsensusValue().Timestamp }
func (x ViewExtBasisBlock) Upgrades() []proto.Upgrade  { return x.Ext.ConsensusValue().Upgrades }
func (x ViewExtBasisBlock) PrevValue() proto.ConsensusValueHash {
	return x.Ext.ConsensusValue().Prev_value_hash
}
func (x ViewExtBasisBlock) IsMissing() bool  { return false }
func (x ViewExtBasisBlock) IsExtBasis() bool { return true }
func (x ViewExtBasisBlock) IsExtValue() bool { return false }
func (x ViewExtBasisBlock) IsCatchup() bool  { return false }
func (x ViewExtBasisBlock) IsDb() bool       { return false }
func (x ViewExtBasisBlock) Committable() batonkit.Committable {
	switch {
	case x.Validated != nil:
		return batonkit.NewCommittableNomination(
			x.Ext.ConsensusBasis(),
			x.Ext.ConsensusValue(),
			x.Validated,
		)
	case x.Catchup != nil:
		return batonkit.NewCommittableCatchup(
			x.Catchup.Block(),
			x.Catchup.ConsensusValue(),
			x.Catchup.SignedTransactions(),
			nil,
		)
	}
	return nil
}
func (x ViewExtBasisBlock) Icon() string {
	return fmt.Sprintf("X:%d", x.Ext.Height())
}

// externalized value block

type ViewExtValueBlock struct {
	Ext     externalizedValue
	Catchup *CatchupArrival
}

func (x ViewExtValueBlock) Proposer() proto.ValidatorID { return x.Ext.Proposer() }
func (x ViewExtValueBlock) BlockHash() proto.BlockHash {
	return x.Ext.BlockHash()
}
func (x ViewExtValueBlock) BlockHeight() proto.BlockHeight {
	return x.Ext.Height()
}
func (x ViewExtValueBlock) PrevBlockHash() proto.BlockHash {
	return x.Ext.PrevBlockHash()
}
func (x ViewExtValueBlock) BasisHash() proto.ConsensusBasisHash {
	return x.Ext.ConsensusBasisHash()
}
func (x ViewExtValueBlock) PrevBasisHash() proto.ConsensusBasisHash {
	return x.Ext.PrevConsensusBasisHash()
}
func (x ViewExtValueBlock) Timestamp() proto.TimePoint { return x.Ext.ConsensusValue().Timestamp }
func (x ViewExtValueBlock) Upgrades() []proto.Upgrade  { return x.Ext.ConsensusValue().Upgrades }
func (x ViewExtValueBlock) PrevValue() proto.ConsensusValueHash {
	return x.Ext.ConsensusValue().Prev_value_hash
}
func (x ViewExtValueBlock) IsMissing() bool  { return false }
func (x ViewExtValueBlock) IsExtBasis() bool { return false }
func (x ViewExtValueBlock) IsExtValue() bool { return true }
func (x ViewExtValueBlock) IsCatchup() bool  { return false }
func (x ViewExtValueBlock) IsDb() bool       { return false }
func (x ViewExtValueBlock) Committable() batonkit.Committable {
	if x.Catchup != nil {
		return batonkit.NewCommittableCatchup(
			x.Catchup.Block(),
			x.Catchup.ConsensusValue(),
			x.Catchup.SignedTransactions(),
			nil,
		)
	}
	return nil
}
func (x ViewExtValueBlock) Icon() string {
	return fmt.Sprintf("Y:%d", x.Ext.Height())
}

// missing block

type ViewMissing struct {
	Height proto.BlockHeight
}

func (x ViewMissing) Proposer() proto.ValidatorID             { panic("must not call") }
func (x ViewMissing) BlockHash() proto.BlockHash              { panic("must not call") }
func (x ViewMissing) BlockHeight() proto.BlockHeight          { return x.Height }
func (x ViewMissing) PrevBlockHash() proto.BlockHash          { panic("must not call") }
func (x ViewMissing) BasisHash() proto.ConsensusBasisHash     { panic("must not call") }
func (x ViewMissing) PrevBasisHash() proto.ConsensusBasisHash { panic("must not call") }
func (x ViewMissing) Timestamp() proto.TimePoint              { panic("must not call") }
func (x ViewMissing) Upgrades() []proto.Upgrade               { panic("must not call") }
func (x ViewMissing) PrevValue() proto.ConsensusValueHash     { panic("must not call") }
func (x ViewMissing) IsMissing() bool                         { return true }
func (x ViewMissing) IsExtBasis() bool                        { return false }
func (x ViewMissing) IsExtValue() bool                        { return false }
func (x ViewMissing) IsCatchup() bool                         { return false }
func (x ViewMissing) IsDb() bool                              { return false }
func (x ViewMissing) Committable() batonkit.Committable       { return nil }
func (x ViewMissing) Icon() string                            { return fmt.Sprintf("?:%d", x.Height) }
