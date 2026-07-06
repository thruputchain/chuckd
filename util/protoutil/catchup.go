package protoutil

import (
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// VerifyCatchupBlock verifies the integrity of the catchup block:
// the included signed transactions match the signed transaction hashes in the consensus basis
func VerifyCatchupBlock(cb *proto.CatchupBlock) (
	ProposalTxsSerialized,
	[]proto.SignedTransaction,
	bool,
) {

	txh, err := xdrutil.SerializeSignedTransactionHashBatch(
		cb.Consensus_basis.Proposal_txs.Hash)
	if err != nil {
		return ProposalTxsSerialized{}, nil, false
	}

	txd, err := xdrutil.SerializeNondeterministicProposalDataBatch(
		cb.Consensus_basis.Proposal_txs.Nondeterministic_data)
	if err != nil {
		return ProposalTxsSerialized{}, nil, false
	}

	txs, err := xdrutil.SerializeSignedTransactionBatch(cb.Signed_transactions)
	if err != nil {
		return ProposalTxsSerialized{}, nil, false
	}

	if len(txh) != len(txs) || len(txh) != len(txd) {
		return ProposalTxsSerialized{}, nil, false
	}

	for i, txh := range txh {
		if txh != HashSignedTx(&txs[i]) {
			return ProposalTxsSerialized{}, nil, false
		}
	}
	return ProposalTxsSerialized{
		Txs:              &cb.Consensus_basis.Proposal_txs,
		Transaction_hash: txh,
		Transaction_data: txd,
	}, txs, true
}
