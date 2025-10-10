package baton

import (
	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type ViewBlock interface {
	BlockHash() proto.BlockHash
	BlockHeight() proto.BlockHeight
	PrevBlockHash() proto.BlockHash
	IsReadyToValidate() bool // enough data is available to run block validation
	IsMissing() bool
	IsSCP() bool
	IsCatchup() bool
	IsDb() bool
	ValidatableBlock() *batonapi.ValidatableBlock
	Icon() string
}

type ViewDbBlock struct {
	Tip batonapi.Tip
}

func (x ViewDbBlock) BlockHash() proto.BlockHash                   { return x.Tip.Hash }
func (x ViewDbBlock) BlockHeight() proto.BlockHeight               { return x.Tip.Height }
func (x ViewDbBlock) PrevBlockHash() proto.BlockHash               { panic("should not be called") }
func (x ViewDbBlock) IsReadyToValidate() bool                      { panic("should not be called") }
func (x ViewDbBlock) IsMissing() bool                              { return false }
func (x ViewDbBlock) IsSCP() bool                                  { return false }
func (x ViewDbBlock) IsCatchup() bool                              { return false }
func (x ViewDbBlock) IsDb() bool                                   { return true }
func (x ViewDbBlock) ValidatableBlock() *batonapi.ValidatableBlock { panic("should not be called") }
func (x ViewDbBlock) Icon() string                                 { return "D" }

type ViewCatchupBlock struct {
	Linked    bool // all catchup blocks between db and first scp chain hashes correctly
	Candidate CatchupBlock
}

func (x ViewCatchupBlock) BlockHash() proto.BlockHash     { return x.Candidate.BlockHash() }
func (x ViewCatchupBlock) BlockHeight() proto.BlockHeight { return x.Candidate.BlockHeight() }
func (x ViewCatchupBlock) PrevBlockHash() proto.BlockHash { return x.Candidate.PrevBlockHash() }
func (x ViewCatchupBlock) IsReadyToValidate() bool        { return x.Linked }
func (x ViewCatchupBlock) IsMissing() bool                { return false }
func (x ViewCatchupBlock) IsSCP() bool                    { return false }
func (x ViewCatchupBlock) IsCatchup() bool                { return true }
func (x ViewCatchupBlock) IsDb() bool                     { return false }
func (x ViewCatchupBlock) ValidatableBlock() *batonapi.ValidatableBlock {
	return &batonapi.ValidatableBlock{
		FromCatchup:          true,
		BlockHeight:          x.BlockHeight(),
		BlockHash:            x.BlockHash(),
		PrevBlockHash:        x.PrevBlockHash(),
		ProposedTransactions: x.Candidate.Txs,
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
	Nomination *proto.Nomination
	Txs        []*proto.SignedTransaction
	Catchup    *CatchupBlock // a catchup block can be used if txs are missing
}

func (x ViewExternalizedBlock) BlockHash() proto.BlockHash     { return x.Nomination.Block_hash }
func (x ViewExternalizedBlock) BlockHeight() proto.BlockHeight { return x.Nomination.Block_height }
func (x ViewExternalizedBlock) PrevBlockHash() proto.BlockHash { return x.Nomination.Prev_block_hash }
func (x ViewExternalizedBlock) IsReadyToValidate() bool {
	// if a catchup block is placed inside an scp block, then it must link against it by design
	return (!x.IsMissingTransactions()) || x.Catchup != nil
}
func (x ViewExternalizedBlock) IsMissingTransactions() bool {
	return len(x.Txs) != len(x.Nomination.Transaction_hashes)
}
func (x ViewExternalizedBlock) IsMissing() bool { return false }
func (x ViewExternalizedBlock) IsSCP() bool     { return true }
func (x ViewExternalizedBlock) IsCatchup() bool { return false }
func (x ViewExternalizedBlock) IsDb() bool      { return false }
func (x ViewExternalizedBlock) ValidatableBlock() *batonapi.ValidatableBlock {
	var txs []*proto.ProposedTransaction
	var fromCatchup bool
	var fromConsensus bool
	if x.IsMissingTransactions() {
		fromCatchup = true
		txs = protoutil.UnpackProposedTransactions(x.Catchup.Block.Proposed_transactions)
	} else {
		fromConsensus = true
		txs = make([]*proto.ProposedTransaction, len(x.Nomination.Transaction_hashes))
		for i := range txs {
			txs[i] = &proto.ProposedTransaction{
				Signed_tx:                      *x.Txs[i],
				Nondeterministic_proposal_data: x.Nomination.Transaction_nominations[i].Nondeterministic_proposal_data,
			}
		}
	}
	return &batonapi.ValidatableBlock{
		FromCatchup:          fromCatchup,
		FromConsensus:        fromConsensus,
		BlockHeight:          x.BlockHeight(),
		BlockHash:            x.BlockHash(),
		PrevBlockHash:        x.PrevBlockHash(),
		ProposedTransactions: txs,
	}
}
func (x ViewExternalizedBlock) Icon() string {
	return "X"
}

type ViewMissing struct {
	Height proto.BlockHeight
}

func (x ViewMissing) BlockHash() proto.BlockHash                   { panic("must not call") }
func (x ViewMissing) BlockHeight() proto.BlockHeight               { return x.Height }
func (x ViewMissing) PrevBlockHash() proto.BlockHash               { panic("must not call") }
func (x ViewMissing) IsReadyToValidate() bool                      { return false }
func (x ViewMissing) IsMissing() bool                              { return true }
func (x ViewMissing) IsSCP() bool                                  { return false }
func (x ViewMissing) IsCatchup() bool                              { return false }
func (x ViewMissing) IsDb() bool                                   { return false }
func (x ViewMissing) ValidatableBlock() *batonapi.ValidatableBlock { panic("must not call") }
func (x ViewMissing) Icon() string                                 { return "?" }
