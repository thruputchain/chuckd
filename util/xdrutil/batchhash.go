package xdrutil

import (
	"crypto/sha256"
	"sync"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// parallelBatchDomainSeparator is cxs(DomainSeparator("ParallelBatch")): the
// zero ExtensionPoint, the length of the name, and the name.
var parallelBatchDomainSeparator = []byte("\x00\x0dParallelBatch")

// HashParallelBatch commits to the index of a batch and to each of its
// segments, in order:
//
//	SHA-256(cxs(DomainSeparator("ParallelBatch")) || SHA-256(index) ||
//	        SHA-256(segment_0) || ... || SHA-256(segment_{n-1}))
//
// where the index is the XDR of the batch with its bytes removed. The segments
// hash in parallel. Combining their hashes in order, rather than by XOR, binds
// each segment to its position, so no other batch of the same shape can be
// built to hash alike (hogd#4508).
//
// batch must be pre-validated with ValidateBatch, otherwise HashParallelBatch will panic.
func HashParallelBatch(batch proto.ParallelBatch) proto.SHA256Hash {

	segHashes := make([][sha256.Size]byte, len(batch.Segment_bytes))
	var wg sync.WaitGroup
	wg.Add(len(segHashes))
	offset := 0
	for i, k := range batch.Segment_bytes {
		go func(segment []byte) {
			defer wg.Done()
			segHashes[i] = sha256.Sum256(segment)
		}(batch.Xdr[offset : offset+int(k)])
		offset += int(k)
	}

	index := proto.ParallelBatch{
		Segment_tokens: batch.Segment_tokens,
		Segment_bytes:  batch.Segment_bytes,
	}
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, &index)
	indexHash := h.Sum(nil)

	wg.Wait()
	h.Reset()
	h.Write(parallelBatchDomainSeparator)
	h.Write(indexHash)
	for i := range segHashes {
		h.Write(segHashes[i][:])
	}
	return proto.SHA256Hash(h.Sum(nil))
}
