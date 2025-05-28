package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
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
	return sha256.Sum256(xdrutil.MarshalXdr(tx))
}
