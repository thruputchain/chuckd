package baton

import (
	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	proto "github.com/groundhogchain/protocol-go"
)

type ViewBlock interface {
	BlockHash() proto.BlockHash
	BlockHeight() proto.BlockHeight
	PrevBlockHash() proto.BlockHash
	ConsensusValueHash() proto.ConsensusValueHash
	PrevConsensusValueHash() proto.ConsensusValueHash
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

// db block

type ViewDbBlock struct {
	TipDb  proto.DbTip
	TipExt externalizedConsensus
}

func (x ViewDbBlock) BlockHash() proto.BlockHash     { return x.TipDb.Hash }
func (x ViewDbBlock) BlockHeight() proto.BlockHeight { return x.TipDb.Height }
func (x ViewDbBlock) PrevBlockHash() proto.BlockHash { panic("should not be called") }
func (x ViewDbBlock) ConsensusValueHash() proto.ConsensusValueHash {
	return x.TipExt.ConsensusValueHash()
}
func (x ViewDbBlock) PrevConsensusValueHash() proto.ConsensusValueHash {
	return x.TipExt.ConsensusValue().Prev_cv_hash
}
func (x ViewDbBlock) IsMissing() bool                   { return false }
func (x ViewDbBlock) IsExtConsensus() bool              { return false }
func (x ViewDbBlock) IsExtCue() bool                    { return false }
func (x ViewDbBlock) IsCatchup() bool                   { return false }
func (x ViewDbBlock) IsDb() bool                        { return true }
func (x ViewDbBlock) Committable() batonkit.Committable { panic("should not be called") }
func (x ViewDbBlock) Icon() string                      { return "D" }

// catchup block

type ViewCatchupBlock struct {
	Linked    bool // all catchup blocks between db and first scp chain hashes correctly
	Candidate CatchupArrival
	Validated *proto.BlockID
}

func (x ViewCatchupBlock) BlockHash() proto.BlockHash     { return x.Candidate.BlockHash() }
func (x ViewCatchupBlock) BlockHeight() proto.BlockHeight { return x.Candidate.BlockHeight() }
func (x ViewCatchupBlock) PrevBlockHash() proto.BlockHash { return x.Candidate.PrevBlockHash() }
func (x ViewCatchupBlock) ConsensusValueHash() proto.ConsensusValueHash {
	return x.Candidate.ConsensusValueHash()
}
func (x ViewCatchupBlock) PrevConsensusValueHash() proto.ConsensusValueHash {
	return x.Candidate.ConsensusValue().Prev_cv_hash
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
		x.Candidate.ConsensusValueHash(),
		x.Candidate.SignedTransactions(),
		x.Validated,
	)
}
func (x ViewCatchupBlock) Icon() string {
	if x.Linked {
		return "L"
	} else {
		return "C"
	}
}

// externalized consensus block

type ViewExtConsensusBlock struct {
	Ext       externalizedConsensus
	Validated *proto.BlockID
	Catchup   *CatchupArrival
}

func (x ViewExtConsensusBlock) BlockHash() proto.BlockHash {
	return x.Ext.BlockHash()
}
func (x ViewExtConsensusBlock) BlockHeight() proto.BlockHeight {
	return x.Ext.Height()
}
func (x ViewExtConsensusBlock) PrevBlockHash() proto.BlockHash {
	return x.Ext.PrevBlockHash()
}
func (x ViewExtConsensusBlock) ConsensusValueHash() proto.ConsensusValueHash {
	return x.Ext.ConsensusValueHash()
}
func (x ViewExtConsensusBlock) PrevConsensusValueHash() proto.ConsensusValueHash {
	return x.Ext.PrevConsensusValueHash()
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
			x.Ext.ConsensusValue(),
			x.Ext.ConsensusValueHash(),
			x.Validated,
		)
	case x.Catchup != nil:
		return batonkit.NewCommittableCatchup(
			x.Catchup.Block(),
			x.Catchup.ConsensusValueHash(),
			x.Catchup.SignedTransactions(),
			nil,
		)
	}
	return nil
}
func (x ViewExtConsensusBlock) Icon() string {
	return "X"
}

// externalized cue block

type ViewExtCueBlock struct {
	Ext     externalizedCue
	Catchup *CatchupArrival
}

func (x ViewExtCueBlock) BlockHash() proto.BlockHash {
	return x.Ext.BlockHash()
}
func (x ViewExtCueBlock) BlockHeight() proto.BlockHeight {
	return x.Ext.Height()
}
func (x ViewExtCueBlock) PrevBlockHash() proto.BlockHash {
	return x.Ext.PrevBlockHash()
}
func (x ViewExtCueBlock) ConsensusValueHash() proto.ConsensusValueHash {
	return x.Ext.ConsensusValueHash()
}
func (x ViewExtCueBlock) PrevConsensusValueHash() proto.ConsensusValueHash {
	return x.Ext.PrevConsensusValueHash()
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
			x.Catchup.ConsensusValueHash(),
			x.Catchup.SignedTransactions(),
			nil,
		)
	}
	return nil
}
func (x ViewExtCueBlock) Icon() string {
	return "Y"
}

// missing block

type ViewMissing struct {
	Height proto.BlockHeight
}

func (x ViewMissing) BlockHash() proto.BlockHash                       { panic("must not call") }
func (x ViewMissing) BlockHeight() proto.BlockHeight                   { return x.Height }
func (x ViewMissing) PrevBlockHash() proto.BlockHash                   { panic("must not call") }
func (x ViewMissing) ConsensusValueHash() proto.ConsensusValueHash     { panic("must not call") }
func (x ViewMissing) PrevConsensusValueHash() proto.ConsensusValueHash { panic("must not call") }
func (x ViewMissing) IsMissing() bool                                  { return true }
func (x ViewMissing) IsExtConsensus() bool                             { return false }
func (x ViewMissing) IsExtCue() bool                                   { return false }
func (x ViewMissing) IsCatchup() bool                                  { return false }
func (x ViewMissing) IsDb() bool                                       { return false }
func (x ViewMissing) Committable() batonkit.Committable                { return nil }
func (x ViewMissing) Icon() string                                     { return "?" }
