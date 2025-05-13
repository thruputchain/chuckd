package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func PrepareBatch(ptxs []proto.ProposedTransaction) (proto.BatchTransactions, proto.BatchHash) {
	batch := proto.BatchTransactions{
		Transactions: ptxs,
	}
	return batch, HashBatch(&batch)
}

func HashBatch(batch *proto.BatchTransactions) proto.BatchHash {
	return sha256.Sum256(xdrutil.MarshalXdr(batch))
}
