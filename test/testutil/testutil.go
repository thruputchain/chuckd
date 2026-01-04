package testutil

import (
	"context"
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
		Signatures: []proto.DecoratedSignatureList{
			{
				List: []proto.DecoratedSignature{{
					Public_key_id: 0,
					Signature:     protoutil.SignXdr(h.m, h.user, &tx),
				}},
			},
		},
		Tx: tx,
	}
}

var TestNondeterministicData = proto.NondeterministicProposalData{
	Nondeterministic_bits:      []byte{0xff},
	Nondeterministic_bit_count: 8,
}

type TestBlock struct {
	ConsensusValue    proto.ConsensusValue
	ConsensusValueCue proto.ConsensusValueCue
	Txs               []proto.SignedTransaction
	Catchup           proto.CatchupBlock
}

type BlockHandle struct {
	Hash   proto.BlockHash
	Height proto.BlockHeight
	CV     proto.ConsensusValue
}

func BlockHandleFromTip(tipDb proto.DbTip, tipCV proto.ConsensusValue) *BlockHandle {
	return &BlockHandle{Hash: tipDb.Hash, Height: tipDb.Height, CV: tipCV}
}

func BlockHandleFromCommittableBlock(cb *batonkit.CommittableBlock) *BlockHandle {
	return &BlockHandle{
		Hash:   cb.BlockHash(),
		Height: cb.BlockHeight(),
		CV:     *cb.AsConsensusValue(),
	}
}

func (h *TestGen) GenNominationOntoTip(
	ctx context.Context,
	ntxs int,
	ontoDb proto.DbTip,
	ontoCV proto.ConsensusValue,
) *TestBlock {
	return h.GenNominationOnto(ctx, ntxs, BlockHandleFromTip(ontoDb, ontoCV))
}

func (h *TestGen) GenNominationOntoBlock(ctx context.Context, ntxs int, onto *TestBlock) *TestBlock {
	return h.GenNominationOnto(
		ctx,
		ntxs,
		&BlockHandle{
			Height: onto.ConsensusValue.Nomination.Block_height,
			Hash:   onto.ConsensusValue.Nomination.Block_hash,
			CV:     onto.ConsensusValue,
		},
	)
}

func (h *TestGen) GenNominationOnto(ctx context.Context, numTxs int, onto *BlockHandle) *TestBlock {
	txs := make([]proto.SignedTransaction, numTxs)
	ntxs := make([]proto.NondeterministicProposalData, numTxs)
	ptxs := make([]proto.ProposedTransaction, numTxs)
	hashes := make([]proto.SignedTransactionHash, numTxs)
	hash := onto.Hash
	for i := range txs {
		txs[i] = *h.GenSignedTx(ctx)
		hashes[i] = protoutil.HashSignedTx(h.m, &txs[i])
		ntxs[i] = TestNondeterministicData
		ptxs[i] = proto.ProposedTransaction{
			Signed_tx:                      txs[i],
			Nondeterministic_proposal_data: TestNondeterministicData,
		}
		hash = BlendPropopsedTx(h.m, hash, &ptxs[i])
	}
	cv := proto.ConsensusValue{
		Timestamp: protoutil.TimepointFromTime(h.time.Now()),
		Nomination: proto.Nomination{
			Proposer:        h.validator.Public_info,
			Block_height:    onto.Height + 1,
			Block_hash:      hash,
			Prev_block_hash: onto.Hash,
			Transactions_hash: xdrutil.MarshalSignedTransactionHashBatch(
				h.m, systemfixed.NominationSignedTransactionHashBatchMaxSegments, hashes),
			Transactions_nondeterministic_data: xdrutil.MarshalNondeterministicProposalDataBatch(
				h.m, systemfixed.NominationNondeterministicDataBatchMaxSegments, ntxs),
		},
		Upgrades:     nil, // XXX: upgrades not yet supported in test
		Prev_cv_hash: protoutil.HashCV(h.m, &onto.CV),
	}
	return &TestBlock{
		ConsensusValue:    cv,
		ConsensusValueCue: protoutil.CueFromConsensusValue(h.m, &cv),
		Txs:               txs,
		Catchup: proto.CatchupBlock{
			Consensus_value: cv,
			Signed_transactions: xdrutil.MarshalSignedTransactionBatch(
				h.m, systemfixed.CatchupTxBatchMaxSegments, txs),
		},
	}
}

func BlendPropopsedTx(m libxdr.XdrMarshaler, base proto.BlockHash, ptx *proto.ProposedTransaction) proto.BlockHash {
	return xorBlockHash(base, protoutil.HashSignedTx(m, &ptx.Signed_tx))
}

func BlendSignedTx(m libxdr.XdrMarshaler, base proto.BlockHash, stx *proto.SignedTransaction) proto.BlockHash {
	return xorBlockHash(base, protoutil.HashSignedTx(m, stx))
}

func xorBlockHash(p, q proto.BlockHash) proto.BlockHash {
	var r proto.BlockHash
	for i := range r {
		r[i] = p[i] ^ q[i]
	}
	return r
}
