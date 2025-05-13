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

// GenProposal generates a proposal onto the current db tip.
func (h *TestRig) GenProposal(ctx context.Context, ntxs int) (*proto.Proposal, *proto.BatchTransactions) {
	onto := h.TestDb.GetTip()
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
	return &proto.Proposal{
		Block_height:                 onto.Height + 1,
		Block_hash:                   hash,
		Prev_block_hash:              onto.Hash,
		Proposed_transaction_batches: []proto.BatchHash{batchHash},
	}, &batch
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
