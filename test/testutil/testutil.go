package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"sync"

	"github.com/thruputchain/chuckd/sub/baton/batonkit"
	"github.com/thruputchain/chuckd/system/systemfixed"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/chuckd/util/xdrutil"
	"github.com/thruputchain/lib-go/libtime"
	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
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

// TestNetworkHash is the network hash the mock scheduler places in proposal headers.
var TestNetworkHash = proto.NetworkHash(sha256.Sum256([]byte("testnet")))

// TestEventsHash is the events hash the mock scheduler places in proposal headers.
// The mock scheduler does not compute events.
var TestEventsHash = proto.EventsHash{}

type TestBlock struct {
	ConsensusBasis proto.ConsensusBasis
	ConsensusValue proto.ConsensusValue
	Txs            []proto.SignedTransaction
	Catchup        proto.CatchupBlock
}

type BlockHandle struct {
	Height      proto.BlockHeight
	StorageHash proto.StorageShortHash
	Value       proto.ConsensusValue
}

func BlockHandleFromTip(tipDb proto.DbTip) *BlockHandle {
	return &BlockHandle{
		Height:      tipDb.Value.Proposal_header.Block_height,
		StorageHash: tipDb.Value.Proposal_header.Storage_hash,
		Value:       tipDb.Value,
	}
}

func BlockHandleFromCommittableBlock(cb *batonkit.CommittableBlock) *BlockHandle {
	value := *cb.ConsensusValue()
	return &BlockHandle{
		Height:      value.Proposal_header.Block_height,
		StorageHash: value.Proposal_header.Storage_hash,
		Value:       value,
	}
}

func (h *TestGen) GenProposalOntoTip(
	ctx context.Context,
	ntxs int,
	ontoDb proto.DbTip,
) *TestBlock {
	return h.GenProposalOnto(ctx, ntxs, BlockHandleFromTip(ontoDb))
}

func (h *TestGen) GenProposalOntoBlock(ctx context.Context, ntxs int, onto *TestBlock) *TestBlock {
	return h.GenProposalOnto(
		ctx,
		ntxs,
		&BlockHandle{
			Height:      onto.ConsensusValue.Proposal_header.Block_height,
			StorageHash: onto.ConsensusValue.Proposal_header.Storage_hash,
			Value:       onto.ConsensusValue,
		},
	)
}

func (h *TestGen) GenProposalOnto(ctx context.Context, numTxs int, onto *BlockHandle) *TestBlock {
	txs := make([]proto.SignedTransaction, numTxs)
	ntxs := make([]proto.NondeterministicProposalData, numTxs)
	ptxs := make([]proto.ProposedTransaction, numTxs)
	hashes := make([]proto.TransactionHash, numTxs)
	storage := onto.StorageHash
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
	storage = UniqueStorageHash(onto.Height+1, storage)
	proposalTxs := proto.ProposalTxs{
		Signed_hash: xdrutil.ParallelizeSignedTransactionHashBatch(
			h.m, systemfixed.NominationTransactionHashBatchMaxSegments, hashes),
		Nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(
			h.m, systemfixed.NominationNondeterministicDataBatchMaxSegments, ntxs),
	}
	value := proto.ConsensusValue{
		Protocol:  proto.Protocol(0),
		Prev_hash: protoutil.HashValueToProto(&onto.Value),
		Proposal_header: proto.ProposalHeader{
			Network_hash:    TestNetworkHash,
			Proposer:        protoutil.PublicVID(h.validator.Public_info),
			Block_timepoint: protoutil.TimepointFromTime(h.time.Now()),
			Block_height:    onto.Height + 1,
			Txs_hash:        TxsHashOf(hashes),
			Storage_hash:    storage,
			Events_hash:     TestEventsHash,
		},
		Proposal_txs_hash: protoutil.HashProposalTxsFast(&proposalTxs),
		Upgrades:          nil, // XXX: upgrades not yet supported in test
	}
	basis := proto.ConsensusBasis{
		Value:        value,
		Proposal_txs: proposalTxs,
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

func BlendPropopsedTx(base proto.StorageShortHash, ptx *proto.ProposedTransaction) proto.StorageShortHash {
	return xorHash(base, protoutil.HashSignedTx(&ptx.Signed_tx))
}

func BlendSignedTx(base proto.StorageShortHash, stx *proto.SignedTransaction) proto.StorageShortHash {
	return xorHash(base, protoutil.HashSignedTx(stx))
}

func UniqueStorageHash(height proto.BlockHeight, hash proto.StorageShortHash) proto.StorageShortHash {
	var heightBytes [8]byte
	binary.BigEndian.PutUint64(heightBytes[:], uint64(height))
	heightHash := sha256.Sum256(heightBytes[:])
	for i := range hash {
		hash[i] ^= heightHash[i]
	}
	return hash
}

// TxsHashOf computes the mock scheduler's stand-in for the Merkle root hash of the transactions.
func TxsHashOf(hashes []proto.TransactionHash) proto.TxsHash {
	var r proto.TxsHash
	for _, h := range hashes {
		r = xorHash(r, h)
	}
	return r
}

func xorHash(p, q proto.SHA256Hash) proto.SHA256Hash {
	var r proto.SHA256Hash
	for i := range r {
		r[i] = p[i] ^ q[i]
	}
	return r
}
