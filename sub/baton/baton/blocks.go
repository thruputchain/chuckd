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
	IsMissing() bool
	IsExt() bool
	IsCatchup() bool
	IsDb() bool
	Committable() batonkit.Committable
	Icon() string
}

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
func (x ViewDbBlock) IsExt() bool                       { return false }
func (x ViewDbBlock) IsCatchup() bool                   { return false }
func (x ViewDbBlock) IsDb() bool                        { return true }
func (x ViewDbBlock) Committable() batonkit.Committable { panic("should not be called") }
func (x ViewDbBlock) Icon() string                      { return "D" }

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
func (x ViewCatchupBlock) IsMissing() bool { return false }
func (x ViewCatchupBlock) IsExt() bool     { return false }
func (x ViewCatchupBlock) IsCatchup() bool { return true }
func (x ViewCatchupBlock) IsDb() bool      { return false }
func (x ViewCatchupBlock) Committable() batonkit.Committable {
	if !x.Linked {
		return nil
	}
	return &batonkit.CommittableCatchup{
		Catchup:   x.Candidate.Block(),
		Validated: x.Validated,
	}
}
func (x ViewCatchupBlock) Icon() string {
	if x.Linked {
		return "L"
	} else {
		return "C"
	}
}

type ViewExternalizedBlock struct {
	Ext       externalizedConsensus
	Validated *proto.BlockID
	Catchup   *CatchupArrival
}

func (x ViewExternalizedBlock) BlockHash() proto.BlockHash {
	return x.Ext.BlockHash()
}
func (x ViewExternalizedBlock) BlockHeight() proto.BlockHeight {
	return x.Ext.Height()
}
func (x ViewExternalizedBlock) PrevBlockHash() proto.BlockHash {
	return x.Ext.PrevBlockHash()
}
func (x ViewExternalizedBlock) ConsensusValueHash() proto.ConsensusValueHash {
	return x.Ext.ConsensusValueHash()
}
func (x ViewExternalizedBlock) PrevConsensusValueHash() proto.ConsensusValueHash {
	return x.Ext.ConsensusValue().Prev_cv_hash
}
func (x ViewExternalizedBlock) IsMissing() bool { return false }
func (x ViewExternalizedBlock) IsExt() bool     { return true }
func (x ViewExternalizedBlock) IsCatchup() bool { return false }
func (x ViewExternalizedBlock) IsDb() bool      { return false }
func (x ViewExternalizedBlock) Committable() batonkit.Committable {
	switch {
	case x.Validated != nil:
		return &batonkit.CommittableNomination{
			ConsensusValue: x.Ext.ConsensusValue(),
			Validated:      x.Validated,
		}
	case x.Catchup != nil:
		return &batonkit.CommittableCatchup{
			Catchup: x.Catchup.Block(),
		}
	}
	return nil
}
func (x ViewExternalizedBlock) Icon() string {
	return "X"
}

type ViewMissing struct {
	Height proto.BlockHeight
}

func (x ViewMissing) BlockHash() proto.BlockHash                       { panic("must not call") }
func (x ViewMissing) BlockHeight() proto.BlockHeight                   { return x.Height }
func (x ViewMissing) PrevBlockHash() proto.BlockHash                   { panic("must not call") }
func (x ViewMissing) ConsensusValueHash() proto.ConsensusValueHash     { panic("must not call") }
func (x ViewMissing) PrevConsensusValueHash() proto.ConsensusValueHash { panic("must not call") }
func (x ViewMissing) IsMissing() bool                                  { return true }
func (x ViewMissing) IsExt() bool                                      { return false }
func (x ViewMissing) IsCatchup() bool                                  { return false }
func (x ViewMissing) IsDb() bool                                       { return false }
func (x ViewMissing) Committable() batonkit.Committable                { return nil }
func (x ViewMissing) Icon() string                                     { return "?" }
