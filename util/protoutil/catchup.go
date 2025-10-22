package protoutil

import (
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// VerifyCatchupBlock verifies the integrity of the catchup block:
// the included signed transactions match the signed transaction hashes in the consensus value
func VerifyCatchupBlock(txm xdrutil.XdrMarshaler, cb *proto.CatchupBlock) bool {
	if len(cb.Consensus_value.Nomination.Transaction_hashes) != len(cb.Signed_transactions) {
		return false
	}
	for i, txh := range cb.Consensus_value.Nomination.Transaction_hashes {
		if txh != HashSignedTx(txm, &cb.Signed_transactions[i]) {
			return false
		}
	}
	return true
}
