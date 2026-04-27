package protoutil

import (
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// VerifyCatchupBlock verifies the integrity of the catchup block:
// the included signed transactions match the signed transaction hashes in the consensus value
func VerifyCatchupBlock(cb *proto.CatchupBlock) (
	NominationSerialized,
	[]proto.SignedTransaction,
	bool,
) {

	txh, err := xdrutil.SerializeSignedTransactionHashBatch(cb.Consensus_value.Nomination.Transactions_hash)
	if err != nil {
		return NominationSerialized{}, nil, false
	}
	txd, err := xdrutil.SerializeNondeterministicProposalDataBatch(
		cb.Consensus_value.Nomination.Transactions_nondeterministic_data)
	if err != nil {
		return NominationSerialized{}, nil, false
	}

	txs, err := xdrutil.SerializeSignedTransactionBatch(cb.Signed_transactions)
	if err != nil {
		return NominationSerialized{}, nil, false
	}

	if xdrutil.TokenSizeOfBatch(cb.Consensus_value.Nomination.Transactions_hash) != len(txs) {
		return NominationSerialized{}, nil, false
	}
	for i, txh := range txh {
		if txh != HashSignedTx(&txs[i]) {
			return NominationSerialized{}, nil, false
		}
	}
	return NominationSerialized{
		Nomination:       &cb.Consensus_value.Nomination,
		Transaction_hash: txh,
		Transaction_data: txd,
	}, txs, true
}
