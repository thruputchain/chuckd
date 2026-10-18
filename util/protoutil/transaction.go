package protoutil

import (
	"crypto/sha256"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func Pack[T any](ts []*T) []T {
	ps := make([]T, len(ts))
	for i, t := range ts {
		ps[i] = *t
	}
	return ps
}

func Unpack[T any](ts []T) []*T {
	ps := make([]*T, len(ts))
	for i, t := range ts {
		ps[i] = &t
	}
	return ps
}

func PackProposedTransactions(ts []*proto.ProposedTransaction) []proto.ProposedTransaction {
	return Pack(ts)
}

func UnpackProposedTransactions(ts []proto.ProposedTransaction) []*proto.ProposedTransaction {
	return Unpack(ts)
}

// HashSignedTx hashes the whole signed transaction, signatures included, so
// that proposals commit to the exact encoding each transaction executes.
func HashSignedTx(tx *proto.SignedTransaction) proto.TransactionHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, tx)
	return proto.TransactionHash(h.Sum(nil))
}

func HashSignedTxs(txs []proto.SignedTransaction) []proto.TransactionHash {
	sxs := make([]proto.TransactionHash, len(txs))
	for i := range txs {
		sxs[i] = HashSignedTx(&txs[i])
	}
	return sxs
}
