package xdrutil

import (
	"crypto/sha256"
	"sync"

	proto "github.com/thruputchain/thruput-protocol-go"
)

// parallelBatchDomainSeparator is cxs(DomainSeparator("ParallelBatch")): the
// zero ExtensionPoint, the length of the name, and the name.
var parallelBatchDomainSeparator = []byte("\x00\x0dParallelBatch")

// HashParallelBatch computes the hash that commits to a batch:
//
//	SHA-256(cxs(DomainSeparator("ParallelBatch")) ||
//	        cxs(segment_tokens) || cxs(segment_bytes) ||
//	        SHA-256(SHA-256(segment_0) || ... || SHA-256(segment_{n-1})))
//
// where segment_i is the bytes of xdr that segment_bytes[i] counts. The
// segments hash in parallel. Their hashes are hashed in order and the counts
// are encoded with their lengths, so nothing is combined by XOR and no two
// valid batches share a preimage (hogd#4508).
//
// batch must be pre-validated with ValidateBatch. HashParallelBatch does not
// check the shape itself: bytes of xdr past the counted segments are left out
// of the hash, and counts that overrun xdr are not reliably caught.
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

	// A cxs unsigned int takes at most 5 bytes, and so does each length.
	ncounts := len(batch.Segment_tokens) + len(batch.Segment_bytes)
	preimage := make([]byte, 0, len(parallelBatchDomainSeparator)+5*(2+ncounts)+sha256.Size)
	preimage = append(preimage, parallelBatchDomainSeparator...)
	preimage = appendCxsUint32s(preimage, batch.Segment_tokens)
	preimage = appendCxsUint32s(preimage, batch.Segment_bytes)

	wg.Wait()
	h := sha256.New()
	for i := range segHashes {
		h.Write(segHashes[i][:])
	}
	preimage = h.Sum(preimage)
	return proto.SHA256Hash(sha256.Sum256(preimage))
}
