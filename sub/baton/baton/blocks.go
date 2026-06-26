package baton

import (
	"fmt"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	proto "github.com/groundhogchain/protocol-go"
)

type ViewBlock interface {
	Proposer() proto.ValidatorPublicInfo
	BlockHeight() proto.BlockHeight
	BlockHash() proto.BlockHash
	PrevBlockHash() proto.BlockHash
	ConsensusBasisHash() proto.ConsensusBasisHash
	PrevConsensusBasisHash() proto.ConsensusBasisHash
	//
	IsMissing() bool
	IsExtConsensus() bool
	IsExtCue() bool
	IsCatchup() bool
	IsDb() bool
	//
	Committable() batonkit.Committable
	Icon() string
}

func BlockCue(b ViewBlock) *proto.ConsensusValueCue {
	return &proto.ConsensusValueCue{
		Proposer:        b.Proposer(),
		Block_height:    b.BlockHeight(),
		Cv_hash:         b.ConsensusBasisHash(),
		Prev_cv_hash:    b.PrevConsensusBasisHash(),
		Block_hash:      b.BlockHash(),
		Prev_block_hash: b.PrevBlockHash(),
	}
}

// db block

type ViewDbBlock struct {
	TipDb  proto.DbTip
	TipExt externalizedConsensus
}

func (x ViewDbBlock) Proposer() proto.ValidatorPublicInfo { return x.TipExt.Proposer() }
func (x ViewDbBlock) BlockHash() proto.BlockHash          { return x.TipDb.Block_hash }
func (x ViewDbBlock) BlockHeight() proto.BlockHeight      { return x.TipDb.Height }
func (x ViewDbBlock) PrevBlockHash() proto.BlockHash      { panic("should not be called") }
func (x ViewDbBlock) ConsensusBasisHash() proto.ConsensusBasisHash {
	return x.TipExt.ConsensusBasisHash()
}
func (x ViewDbBlock) PrevConsensusBasisHash() proto.ConsensusBasisHash {
	return x.TipExt.ConsensusBasis().Prev_cv_hash
}
func (x ViewDbBlock) IsMissing() bool                   { return false }
func (x ViewDbBlock) IsExtConsensus() bool              { return false }
func (x ViewDbBlock) IsExtCue() bool                    { return false }
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

func (x ViewCatchupBlock) Proposer() proto.ValidatorPublicInfo { return x.Candidate.Proposer() }
func (x ViewCatchupBlock) BlockHash() proto.BlockHash          { return x.Candidate.BlockHash() }
func (x ViewCatchupBlock) BlockHeight() proto.BlockHeight      { return x.Candidate.BlockHeight() }
func (x ViewCatchupBlock) PrevBlockHash() proto.BlockHash      { return x.Candidate.PrevBlockHash() }
func (x ViewCatchupBlock) ConsensusBasisHash() proto.ConsensusBasisHash {
	return x.Candidate.ConsensusBasisHash()
}
func (x ViewCatchupBlock) PrevConsensusBasisHash() proto.ConsensusBasisHash {
	return x.Candidate.ConsensusBasis().Prev_cv_hash
}
func (x ViewCatchupBlock) IsMissing() bool      { return false }
func (x ViewCatchupBlock) IsExtConsensus() bool { return false }
func (x ViewCatchupBlock) IsExtCue() bool       { return false }
func (x ViewCatchupBlock) IsCatchup() bool      { return true }
func (x ViewCatchupBlock) IsDb() bool           { return false }
func (x ViewCatchupBlock) Committable() batonkit.Committable {
	if !x.Linked {
		return nil
	}
	return batonkit.NewCommittableCatchup(
		x.Candidate.Block(),
		x.Candidate.ConsensusValueCue(),
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

type ViewExtConsensusBlock struct {
	Ext       externalizedConsensus
	Validated *proto.BlockID
	Catchup   *CatchupArrival
}

func (x ViewExtConsensusBlock) Proposer() proto.ValidatorPublicInfo { return x.Ext.Proposer() }
func (x ViewExtConsensusBlock) BlockHash() proto.BlockHash {
	return x.Ext.BlockHash()
}
func (x ViewExtConsensusBlock) BlockHeight() proto.BlockHeight {
	return x.Ext.Height()
}
func (x ViewExtConsensusBlock) PrevBlockHash() proto.BlockHash {
	return x.Ext.PrevBlockHash()
}
func (x ViewExtConsensusBlock) ConsensusBasisHash() proto.ConsensusBasisHash {
	return x.Ext.ConsensusBasisHash()
}
func (x ViewExtConsensusBlock) PrevConsensusBasisHash() proto.ConsensusBasisHash {
	return x.Ext.PrevConsensusBasisHash()
}
func (x ViewExtConsensusBlock) IsMissing() bool      { return false }
func (x ViewExtConsensusBlock) IsExtConsensus() bool { return true }
func (x ViewExtConsensusBlock) IsExtCue() bool       { return false }
func (x ViewExtConsensusBlock) IsCatchup() bool      { return false }
func (x ViewExtConsensusBlock) IsDb() bool           { return false }
func (x ViewExtConsensusBlock) Committable() batonkit.Committable {
	switch {
	case x.Validated != nil:
		return batonkit.NewCommittableNomination(
			x.Ext.ConsensusBasis(),
			x.Ext.ConsensusValueCue(),
			x.Validated,
		)
	case x.Catchup != nil:
		return batonkit.NewCommittableCatchup(
			x.Catchup.Block(),
			x.Catchup.ConsensusValueCue(),
			x.Catchup.SignedTransactions(),
			nil,
		)
	}
	return nil
}
func (x ViewExtConsensusBlock) Icon() string {
	return fmt.Sprintf("X:%d", x.Ext.Height())
}

// externalized cue block

type ViewExtCueBlock struct {
	Ext     externalizedCue
	Catchup *CatchupArrival
}

func (x ViewExtCueBlock) Proposer() proto.ValidatorPublicInfo { return x.Ext.Proposer() }
func (x ViewExtCueBlock) BlockHash() proto.BlockHash {
	return x.Ext.BlockHash()
}
func (x ViewExtCueBlock) BlockHeight() proto.BlockHeight {
	return x.Ext.Height()
}
func (x ViewExtCueBlock) PrevBlockHash() proto.BlockHash {
	return x.Ext.PrevBlockHash()
}
func (x ViewExtCueBlock) ConsensusBasisHash() proto.ConsensusBasisHash {
	return x.Ext.ConsensusBasisHash()
}
func (x ViewExtCueBlock) PrevConsensusBasisHash() proto.ConsensusBasisHash {
	return x.Ext.PrevConsensusBasisHash()
}
func (x ViewExtCueBlock) IsMissing() bool      { return false }
func (x ViewExtCueBlock) IsExtConsensus() bool { return false }
func (x ViewExtCueBlock) IsExtCue() bool       { return true }
func (x ViewExtCueBlock) IsCatchup() bool      { return false }
func (x ViewExtCueBlock) IsDb() bool           { return false }
func (x ViewExtCueBlock) Committable() batonkit.Committable {
	if x.Catchup != nil {
		return batonkit.NewCommittableCatchup(
			x.Catchup.Block(),
			x.Catchup.ConsensusValueCue(),
			x.Catchup.SignedTransactions(),
			nil,
		)
	}
	return nil
}
func (x ViewExtCueBlock) Icon() string {
	return fmt.Sprintf("Y:%d", x.Ext.Height())
}

// missing block

type ViewMissing struct {
	Height proto.BlockHeight
}

func (x ViewMissing) Proposer() proto.ValidatorPublicInfo              { panic("must not call") }
func (x ViewMissing) BlockHash() proto.BlockHash                       { panic("must not call") }
func (x ViewMissing) BlockHeight() proto.BlockHeight                   { return x.Height }
func (x ViewMissing) PrevBlockHash() proto.BlockHash                   { panic("must not call") }
func (x ViewMissing) ConsensusBasisHash() proto.ConsensusBasisHash     { panic("must not call") }
func (x ViewMissing) PrevConsensusBasisHash() proto.ConsensusBasisHash { panic("must not call") }
func (x ViewMissing) IsMissing() bool                                  { return true }
func (x ViewMissing) IsExtConsensus() bool                             { return false }
func (x ViewMissing) IsExtCue() bool                                   { return false }
func (x ViewMissing) IsCatchup() bool                                  { return false }
func (x ViewMissing) IsDb() bool                                       { return false }
func (x ViewMissing) Committable() batonkit.Committable                { return nil }
func (x ViewMissing) Icon() string                                     { return fmt.Sprintf("?:%d", x.Height) }
