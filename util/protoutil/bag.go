package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func PrepareBag(txs []proto.SignedTransaction) (proto.BagTransactions, proto.BagHash) {
	bag := proto.BagTransactions{
		Transactions: txs,
	}
	return bag, HashBag(&bag)
}

func HashBag(bag *proto.BagTransactions) proto.BagHash {
	return sha256.Sum256(xdrutil.MarshalXdr(bag))
}

func ExtractBagSet(bag *proto.BagTransactions, set proto.BagSet) []*proto.SignedTransaction {
	bitmap := set.Bag_bitmap[:min(len(set.Bag_bitmap), divideRoundUpInt(len(bag.Transactions), 8))]
	indices := bitmapToIndices(bitmap, len(bag.Transactions))
	txs := make([]*proto.SignedTransaction, len(indices))
	for i, index := range indices {
		txs[i] = &bag.Transactions[index]
	}
	return txs
}

func divideRoundUpInt(a, b int) int {
	return (a + b - 1) / b
}

func bitmapToIndices(bitmap []byte, limit int) []int {
	var indices []int

	for byteIndex, b := range bitmap {
		for bitIndex := 0; bitIndex < 8; bitIndex++ {
			if b&(1<<bitIndex) != 0 {
				absoluteIndex := byteIndex*8 + bitIndex
				if absoluteIndex < limit {
					indices = append(indices, absoluteIndex)
				}
			}
		}
	}

	return indices
}

func indicesToBitmap(indices []int) []byte {
	if len(indices) == 0 {
		return []byte{}
	}

	maxIndex := 0
	for _, idx := range indices {
		if idx > maxIndex {
			maxIndex = idx
		}
	}

	numBytes := (maxIndex / 8) + 1
	bitmap := make([]byte, numBytes)

	for _, idx := range indices {
		byteIndex := idx / 8
		bitIndex := idx % 8
		bitmap[byteIndex] |= (1 << bitIndex)
	}

	return bitmap
}
