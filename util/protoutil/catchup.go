package protoutil

import (
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

// VerifyCatchupBlock verifies the integrity of the catchup block:
// the included signed transactions match the signed transaction hashes in the consensus value
func VerifyCatchupBlock(txm libxdr.XdrMarshaler, cb *proto.CatchupBlock) bool {

	// XXX(perf): decode batch upstream for perf
	txs, err := xdrutil.UnmarshalSignedTransactionBatch(cb.Signed_transactions)
	if err != nil {
		return false
	}
	// XXX(perf): decode batch upstream for perf
	txhs, err := xdrutil.UnmarshalSignedTransactionHashBatch(cb.Consensus_value.Nomination.Transactions_hash)
	if err != nil {
		return false
	}

	if xdrutil.BatchSize(cb.Consensus_value.Nomination.Transactions_hash) != len(txs) {
		return false
	}
	for i, txh := range txhs {
		if txh != HashSignedTx(txm, &txs[i]) {
			return false
		}
	}
	return true
}
