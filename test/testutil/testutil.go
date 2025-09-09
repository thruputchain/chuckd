package testutil

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestGen struct {
	time libtime.Time
	user proto.ValidatorSecretKey // wallet user who signs txs
	m    xdrutil.XdrMarshaler
	lk   sync.Mutex
	rand *rand.Rand
}

func NewTestGen(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	user proto.ValidatorSecretKey,
) *TestGen {
	return &TestGen{time: time, user: user, m: xdrutil.NewXdrMarshaler(), rand: rand}
}

func (h *TestGen) GenSignedTx(ctx context.Context) *proto.SignedTransaction {
	h.lk.Lock()
	defer h.lk.Unlock()

	var id proto.OriginID
	h.rand.Read(id[:])

	tx := proto.Transaction{
		Origin_id: id,
	}
	return &proto.SignedTransaction{
		Signatures: []proto.DecoratedSignature{
			{Public_key_id: 0, Signature: protoutil.SignXdr(h.m, h.user, &tx)},
		},
		Tx: tx,
	}
}

var TestNondeterministicData = proto.NondeterministicProposalData{
	Nondeterministic_bits:      []byte{0xff},
	Nondeterministic_bit_count: 8,
}

type TestBlock struct {
	Nomination proto.Nomination
	Txs        []proto.SignedTransaction
	Catchup    proto.CatchupBlock
}

type BlockHandle struct {
	Hash   proto.BlockHash
	Height proto.BlockHeight
}

func VBHandle(vb *batonapi.ValidatableBlock) *BlockHandle {
	return &BlockHandle{Hash: vb.BlockHash, Height: vb.BlockHeight}
}

func TipHandle(tip batonapi.Tip) *BlockHandle {
	return &BlockHandle{Hash: tip.Hash, Height: tip.Height}
}

func (h *TestGen) GenNominationOntoTip(ctx context.Context, ntxs int, onto batonapi.Tip) *TestBlock {
	return h.GenNominationOnto(ctx, ntxs, TipHandle(onto))
}

func (h *TestGen) GenNominationOntoVB(ctx context.Context, ntxs int, onto *batonapi.ValidatableBlock) *TestBlock {
	return h.GenNominationOnto(ctx, ntxs, VBHandle(onto))
}

func (h *TestGen) GenNominationOntoBlock(ctx context.Context, ntxs int, onto *TestBlock) *TestBlock {
	return h.GenNominationOnto(ctx, ntxs, &BlockHandle{Height: onto.Nomination.Block_height, Hash: onto.Nomination.Block_hash})
}

func (h *TestGen) GenNominationOnto(ctx context.Context, numTxs int, onto *BlockHandle) *TestBlock {
	txs := make([]proto.SignedTransaction, numTxs)
	ntxs := make([]proto.NominatedTransaction, numTxs)
	ptxs := make([]proto.ProposedTransaction, numTxs)
	hashes := make([]proto.SignedTransactionHash, numTxs)
	hash := onto.Hash
	for i := range txs {
		txs[i] = *h.GenSignedTx(ctx)
		hashes[i] = protoutil.HashSignedTx(h.m, &txs[i])
		ntxs[i] = proto.NominatedTransaction{Nondeterministic_proposal_data: TestNondeterministicData}
		ptxs[i] = proto.ProposedTransaction{
			Signed_tx:                      txs[i],
			Nondeterministic_proposal_data: TestNondeterministicData,
		}
		hash = BlendPropopsedTx(h.m, hash, &ptxs[i])
	}
	return &TestBlock{
		Nomination: proto.Nomination{
			Block_height:            onto.Height + 1,
			Block_hash:              hash,
			Prev_block_hash:         onto.Hash,
			Transaction_hashes:      hashes,
			Transaction_nominations: ntxs,
		},
		Txs: txs,
		Catchup: proto.CatchupBlock{
			Block_height:          onto.Height + 1,
			Block_hash:            hash,
			Prev_block_hash:       onto.Hash,
			Proposed_transactions: ptxs,
		},
	}
}

func BlendPropopsedTx(m xdrutil.XdrMarshaler, base proto.BlockHash, ptx *proto.ProposedTransaction) proto.BlockHash {
	return xorBlockHash(base, protoutil.HashSignedTx(m, &ptx.Signed_tx))
}

func BlendSignedTx(m xdrutil.XdrMarshaler, base proto.BlockHash, stx *proto.SignedTransaction) proto.BlockHash {
	return xorBlockHash(base, protoutil.HashSignedTx(m, stx))
}

func xorBlockHash(p, q proto.BlockHash) proto.BlockHash {
	var r proto.BlockHash
	for i := range r {
		r[i] = p[i] ^ q[i]
	}
	return r
}
