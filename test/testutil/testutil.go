package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type TestGen struct {
	time      libtime.Time
	validator proto.ValidatorPrivateInfo
	user      proto.ValidatorSecretKey // wallet user who signs txs
	m         libxdr.XdrMarshaler
	lk        sync.Mutex
	rand      *rand.Rand
}

func NewTestGen(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	validator proto.ValidatorPrivateInfo,
	user proto.ValidatorSecretKey,
) *TestGen {
	return &TestGen{
		time:      time,
		validator: validator,
		user:      user,
		m:         libxdr.NewXdrMarshaler(),
		rand:      rand,
	}
}

func (h *TestGen) GenSignedTx(ctx context.Context) *proto.SignedTransaction {
	h.lk.Lock()
	defer h.lk.Unlock()

	tx := proto.Transaction{
		Uniquifier: proto.Uniquifier{Nonce: h.rand.Uint64()},
	}
	return &proto.SignedTransaction{
		Signatures: []proto.DecoratedSignature{{
			Public_key_id: 0,
			Signature:     protoutil.SignXdr(h.m, h.user, &tx),
		}},
		Tx: tx,
	}
}

var TestNondeterministicData = proto.NondeterministicProposalData{
	Nondeterministic_bits:      []byte{0xff},
	Nondeterministic_bit_count: 8,
}

type TestBlock struct {
	ConsensusBasis proto.ConsensusBasis
	ConsensusValue proto.ConsensusValue
	Txs            []proto.SignedTransaction
	Catchup        proto.CatchupBlock
}

type BlockHandle struct {
	BlockHash proto.BlockHash
	Height    proto.BlockHeight
	Basis     proto.ConsensusBasis
}

func BlockHandleFromTip(tipDb proto.DbTip, tipCV proto.ConsensusBasis) *BlockHandle {
	return &BlockHandle{BlockHash: tipDb.Block_hash, Height: tipDb.Height, Basis: tipCV}
}

func BlockHandleFromCommittableBlock(cb *batonkit.CommittableBlock) *BlockHandle {
	return &BlockHandle{
		BlockHash: cb.BlockHash(),
		Height:    cb.BlockHeight(),
		Basis:     *cb.AsConsensusBasis(),
	}
}

func (h *TestGen) GenNominationOntoTip(
	ctx context.Context,
	ntxs int,
	ontoDb proto.DbTip,
	ontoBasis proto.ConsensusBasis,
) *TestBlock {
	return h.GenNominationOnto(ctx, ntxs, BlockHandleFromTip(ontoDb, ontoBasis))
}

func (h *TestGen) GenNominationOntoBlock(ctx context.Context, ntxs int, onto *TestBlock) *TestBlock {
	return h.GenNominationOnto(
		ctx,
		ntxs,
		&BlockHandle{
			Height:    onto.ConsensusBasis.Nomination.Block_height,
			BlockHash: onto.ConsensusBasis.Nomination.Block_hash,
			Basis:     onto.ConsensusBasis,
		},
	)
}

func (h *TestGen) GenNominationOnto(ctx context.Context, numTxs int, onto *BlockHandle) *TestBlock {
	txs := make([]proto.SignedTransaction, numTxs)
	ntxs := make([]proto.NondeterministicProposalData, numTxs)
	ptxs := make([]proto.ProposedTransaction, numTxs)
	hashes := make([]proto.SignedTransactionHash, numTxs)
	hash := onto.BlockHash
	for i := range txs {
		txs[i] = *h.GenSignedTx(ctx)
		hashes[i] = protoutil.HashSignedTx(&txs[i])
		ntxs[i] = TestNondeterministicData
		ptxs[i] = proto.ProposedTransaction{
			Signed_tx:                      txs[i],
			Nondeterministic_proposal_data: TestNondeterministicData,
		}
		hash = BlendPropopsedTx(hash, &ptxs[i])
	}
	hash = UniqueBlockHash(onto.Height+1, hash)
	basis := proto.ConsensusBasis{
		Timestamp: protoutil.TimepointFromTime(h.time.Now()),
		Nomination: proto.Nomination{
			Block_height:    onto.Height + 1,
			Block_hash:      hash,
			Prev_block_hash: onto.BlockHash,
			Transactions_hash: xdrutil.ParallelizeSignedTransactionHashBatch(
				h.m, systemfixed.NominationSignedTransactionHashBatchMaxSegments, hashes),
			Transactions_nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(
				h.m, systemfixed.NominationNondeterministicDataBatchMaxSegments, ntxs),
		},
		Upgrades:        nil, // XXX: upgrades not yet supported in test
		Prev_basis_hash: protoutil.HashBasisFast(&onto.Basis),
	}
	return &TestBlock{
		ConsensusBasis: basis,
		ConsensusValue: *protoutil.BasisValue(&basis),
		Txs:            txs,
		Catchup: proto.CatchupBlock{
			Consensus_basis: basis,
			Signed_transactions: xdrutil.ParallelizeSignedTransactionBatch(
				h.m, systemfixed.CatchupTxBatchMaxSegments, txs),
		},
	}
}

func BlendPropopsedTx(base proto.BlockHash, ptx *proto.ProposedTransaction) proto.BlockHash {
	return xorHash(base, protoutil.HashSignedTx(&ptx.Signed_tx))
}

func BlendSignedTx(base proto.BlockHash, stx *proto.SignedTransaction) proto.BlockHash {
	return xorHash(base, protoutil.HashSignedTx(stx))
}

func UniqueBlockHash(height proto.BlockHeight, hash proto.BlockHash) proto.BlockHash {
	var heightBytes [8]byte
	binary.BigEndian.PutUint64(heightBytes[:], uint64(height))
	heightHash := sha256.Sum256(heightBytes[:])
	for i := range hash {
		hash[i] ^= heightHash[i]
	}
	return hash
}

func xorHash(p, q proto.SHA256Hash) proto.SHA256Hash {
	var r proto.SHA256Hash
	for i := range r {
		r[i] = p[i] ^ q[i]
	}
	return r
}
