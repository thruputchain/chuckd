package xdrutil

import (
	"crypto/sha256"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// batch must be pre-validated with ValidateBatch, otherwise HashParallelBatch will panic.
func HashParallelBatch(batch proto.ParallelBatch) proto.SHA256Hash {

	nseg := len(batch.Segment_bytes)
	segHash := make(chan [32]byte, nseg)

	offset := 0
	for _, k := range batch.Segment_bytes {
		go func(offset int) {
			segHash <- sha256.Sum256(batch.Xdr[offset : offset+int(k)])
		}(offset)
		offset += int(k)
	}
	var result [32]byte
	for range nseg {
		s := <-segHash
		for i := range result {
			result[i] ^= s[i]
		}
	}

	batch.Xdr = nil
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, &batch)
	indexHash := [32]byte(h.Sum(nil))
	for i := range result {
		result[i] ^= indexHash[i]
	}

	return result
}
