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

// BlockHandle describes a committed block onto which a new proposal is built.
type BlockHandle struct {
	Value proto.ConsensusValue
}

func (h *BlockHandle) Height() proto.BlockHeight {
	return h.Value.Proposal_header.Block_height
}

func BlockHandleFromTip(tipDb proto.DbTip) *BlockHandle {
	return &BlockHandle{Value: tipDb.Value}
}

func BlockHandleFromCommittableBlock(cb *batonkit.CommittableBlock) *BlockHandle {
	return &BlockHandle{Value: *cb.ConsensusValue()}
}

func (h *TestGen) GenProposalOntoTip(
	ctx context.Context,
	ntxs int,
	ontoDb proto.DbTip,
) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, BlockHandleFromTip(ontoDb))
}

func (h *TestGen) GenProposalOntoBlock(ctx context.Context, ntxs int, onto *TestBlock) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, &BlockHandle{Value: onto.ConsensusValue})
}

func (h *TestGen) GenProposalOnto(ctx context.Context, numTxs int, onto *BlockHandle) *TestBlock {
	txs := make([]proto.SignedTransaction, numTxs)
	ntxs := make([]proto.NondeterministicProposalData, numTxs)
	ptxs := make([]proto.ProposedTransaction, numTxs)
	hashes := make([]proto.SignedTransactionHash, numTxs)
	storage := onto.Value.Proposal_header.Storage_hash
	for i := range txs {
		txs[i] = *h.GenSignedTx(ctx)
		hashes[i] = protoutil.HashSignedTx(&txs[i])
		ntxs[i] = TestNondeterministicData
		ptxs[i] = proto.ProposedTransaction{
			Signed_tx:                      txs[i],
			Nondeterministic_proposal_data: TestNondeterministicData,
		}
		storage = BlendPropopsedTx(storage, &ptxs[i])
	}
	storage = UniqueStorageHash(onto.Height()+1, storage)
	header := proto.ProposalHeader{
		Network_id:   onto.Value.Proposal_header.Network_id,
		Timestamp:    protoutil.TimepointFromTime(h.time.Now()),
		Block_height: onto.Height() + 1,
		Txs_hash:     TxsHashOf(hashes),
		Storage_hash: storage,
	}
	proposal := proto.Proposal{
		Header: header,
		Transactions: proto.ProposalTxs{
			Hash: xdrutil.ParallelizeSignedTransactionHashBatch(
				h.m, systemfixed.NominationSignedTransactionHashBatchMaxSegments, hashes),
			Nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(
				h.m, systemfixed.NominationNondeterministicDataBatchMaxSegments, ntxs),
		},
	}
	value := proto.ConsensusValue{
		Proposer:        onto.Value.Proposer,
		Upgrades:        nil, // XXX: upgrades not yet supported in test
		Proposal_header: header,
		Proposal_hash:   protoutil.HashProposalFast(&proposal),
		Prev_value_hash: protoutil.HashValueToProto(&onto.Value),
	}
	basis := proto.ConsensusBasis{
		Value:        value,
		Proposal_txs: proposal.Transactions,
	}
	return &TestBlock{
		ConsensusBasis: basis,
		ConsensusValue: value,
		Txs:            txs,
		Catchup: proto.CatchupBlock{
			Consensus_basis: basis,
			Signed_transactions: xdrutil.ParallelizeSignedTransactionBatch(
				h.m, systemfixed.CatchupTxBatchMaxSegments, txs),
		},
	}
}

// TxsHashOf stands in for the Merkle root of the transactions in test blocks.
func TxsHashOf(hashes []proto.SignedTransactionHash) proto.TxsHash {
	h := sha256.New()
	for _, x := range hashes {
		h.Write(x[:])
	}
	return proto.TxsHash(h.Sum(nil))
}

func BlendPropopsedTx(base proto.StorageHash, ptx *proto.ProposedTransaction) proto.StorageHash {
	return xorStorageHash(base, protoutil.HashSignedTx(&ptx.Signed_tx))
}

func BlendSignedTx(base proto.StorageHash, stx *proto.SignedTransaction) proto.StorageHash {
	return xorStorageHash(base, protoutil.HashSignedTx(stx))
}

func UniqueStorageHash(height proto.BlockHeight, hash proto.StorageHash) proto.StorageHash {
	var heightBytes [8]byte
	binary.BigEndian.PutUint64(heightBytes[:], uint64(height))
	heightHash := sha256.Sum256(heightBytes[:])
	return xorStorageHash(hash, heightHash)
}

// xorStorageHash XORs a 32-byte hash into the leading bytes of a storage hash.
func xorStorageHash(p proto.StorageHash, q proto.SHA256Hash) proto.StorageHash {
	for i := range q {
		p[i] ^= q[i]
	}
	return p
}
