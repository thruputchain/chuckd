package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
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

func HashSignedTx(tx *proto.SignedTransaction) proto.SignedTransactionHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, &tx.Tx)
	return proto.SignedTransactionHash(h.Sum(nil))
}

func HashSignedTxs(txs []proto.SignedTransaction) []proto.SignedTransactionHash {
	sxs := make([]proto.SignedTransactionHash, len(txs))
	for i := range txs {
		sxs[i] = HashSignedTx(&txs[i])
	}
	return sxs
}
