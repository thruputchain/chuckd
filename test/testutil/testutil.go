package testutil

import (
	"context"
	"crypto/sha256"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestGen struct {
	time libtime.Time
	user proto.ValidatorSecretKey // wallet user who signs txs
	lk   sync.Mutex
	rand *rand.Rand
}

func NewTestGen(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	user proto.ValidatorSecretKey,
) *TestGen {
	return &TestGen{time: time, user: user, rand: rand}
}

func (h *TestGen) GenSignedTx(ctx context.Context) *proto.SignedTransaction {
	h.lk.Lock()
	defer h.lk.Unlock()

	var id proto.OriginID
	h.rand.Read(id[:])

	tx := proto.Transaction{
		Id:      id,
		Created: h.time.Now().UnixNano(),
	}
	return &proto.SignedTransaction{
		Signature: protoutil.SignXdr(h.user, &tx),
		Tx:        tx,
	}
}

var TestNondeterministicData = proto.NondeterministicProposalData{
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

func VBHandle(vb *sdk.ValidatableBlock) *BlockHandle {
	return &BlockHandle{Hash: vb.BlockHash, Height: vb.BlockHeight}
}

func TipHandle(tip sdk.Tip) *BlockHandle {
	return &BlockHandle{Hash: tip.Hash, Height: tip.Height}
}

func (h *TestGen) GenProposalOntoTip(ctx context.Context, ntxs int, onto sdk.Tip) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, TipHandle(onto))
}

func (h *TestGen) GenProposalOntoVB(ctx context.Context, ntxs int, onto *sdk.ValidatableBlock) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, VBHandle(onto))
}

func (h *TestGen) GenProposalOntoBlock(ctx context.Context, ntxs int, onto *TestBlock) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, &BlockHandle{Height: onto.Proposal.Block_height, Hash: onto.Proposal.Block_hash})
}

// GenProposal generates a proposal onto the current db tip.
func (h *TestGen) GenProposalOnto(ctx context.Context, ntxs int, onto *BlockHandle) *TestBlock {
	txs := make([]proto.ProposedTransaction, ntxs)
	hash := onto.Hash
	for i := range txs {
		txs[i] = proto.ProposedTransaction{
			Signed_tx:                      *h.GenSignedTx(ctx),
			Nondeterministic_proposal_data: TestNondeterministicData,
		}
		hash = BlendPropopsedTx(hash, &txs[i])
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

func BlendPropopsedTx(base proto.BlockHash, ptx *proto.ProposedTransaction) proto.BlockHash {
	return xorBlockHash(base, sha256.Sum256(xdrutil.MarshalXdr(&ptx.Signed_tx)))
}

func BlendSignedTx(base proto.BlockHash, stx *proto.SignedTransaction) proto.BlockHash {
	return xorBlockHash(base, sha256.Sum256(xdrutil.MarshalXdr(stx)))
}

func xorBlockHash(p, q proto.BlockHash) proto.BlockHash {
	var r proto.BlockHash
	for i := range r {
		r[i] = p[i] ^ q[i]
	}
	return r
}
