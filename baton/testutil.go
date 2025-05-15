package baton

import (
	"context"
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (h *TestRig) GenSignedTx(ctx context.Context) *proto.SignedTransaction {
	h.rlk.Lock()
	defer h.rlk.Unlock()

	var id proto.OriginID
	h.rand.Read(id[:])

	tx := proto.Transaction{
		Id:      id,
		Created: h.TestTime.Now().UnixNano(),
	}
	return &proto.SignedTransaction{
		Signature: protoutil.SignXdr(h.user, &tx),
		Tx:        tx,
	}
}

var testNondeterministicData = proto.NondeterministicProposalData{
	Nondeterministic_bits:      []byte{0xff},
	Nondeterministic_bit_count: 8,
}

type TestBlock struct {
	Proposal proto.Proposal
	Batch    proto.BatchTransactions
	Catchup  proto.CatchupBlock
}

type BlockHandle struct {
	Hash   proto.BlockHash
	Height proto.BlockHeight
}

func VBHandle(vb *ValidatableBlock) *BlockHandle {
	return &BlockHandle{Hash: vb.BlockHash, Height: vb.BlockHeight}
}

func TipHandle(tip Tip) *BlockHandle {
	return &BlockHandle{Hash: tip.Hash, Height: tip.Height}
}

func (h *TestRig) GenProposal(ctx context.Context, ntxs int) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, TipHandle(h.TestDb.GetTip()))
}

func (h *TestRig) GenProposalOntoVB(ctx context.Context, ntxs int, onto *ValidatableBlock) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, VBHandle(onto))
}

func (h *TestRig) GenProposalOntoBlock(ctx context.Context, ntxs int, onto *TestBlock) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, &BlockHandle{Height: onto.Proposal.Block_height, Hash: onto.Proposal.Block_hash})
}

// GenProposal generates a proposal onto the current db tip.
func (h *TestRig) GenProposalOnto(ctx context.Context, ntxs int, onto *BlockHandle) *TestBlock {
	txs := make([]proto.ProposedTransaction, ntxs)
	hash := onto.Hash
	for i := range txs {
		txs[i] = proto.ProposedTransaction{
			Signed_tx:                      *h.GenSignedTx(ctx),
			Nondeterministic_proposal_data: testNondeterministicData,
		}
		hash = blendPropopsedTx(hash, &txs[i])
	}
	batch, batchHash := protoutil.PrepareBatch(txs)
	return &TestBlock{
		Proposal: proto.Proposal{
			Block_height:                 onto.Height + 1,
			Block_hash:                   hash,
			Prev_block_hash:              onto.Hash,
			Proposed_transaction_batches: []proto.BatchHash{batchHash},
		},
		Batch: batch,
		Catchup: proto.CatchupBlock{
			Block_height:          onto.Height + 1,
			Block_hash:            hash,
			Prev_block_hash:       onto.Hash,
			Proposed_transactions: txs,
		},
	}
}

func blendPropopsedTx(base proto.BlockHash, ptx *proto.ProposedTransaction) proto.BlockHash {
	return xorBlockHash(base, sha256.Sum256(xdrutil.MarshalXdr(ptx)))
}

func blendSignedTx(base proto.BlockHash, stx *proto.SignedTransaction) proto.BlockHash {
	return xorBlockHash(base, sha256.Sum256(xdrutil.MarshalXdr(stx)))
}

func xorBlockHash(p, q proto.BlockHash) proto.BlockHash {
	var r proto.BlockHash
	for i := range r {
		r[i] = p[i] ^ q[i]
	}
	return r
}
