package xdrutil

import (
	"crypto/sha256"
	"sync"

	proto "github.com/thruputchain/thruput-protocol-go"
)

// parallelBatchDomainSeparator is cxs(DomainSeparator("ParallelBatch")), which
// goes before the BatchCommit in the hash: the zero ExtensionPoint, the length
// of the name, and the name.
var parallelBatchDomainSeparator = []byte("\x00\x0dParallelBatch")

// HashParallelBatch computes the hash that commits to a batch:
//
//	SHA-256(cxs(DomainSeparator("ParallelBatch")) || cxs(BatchCommit))
//
// BatchCommit is a struct of the protocol's parallel.x. Its fields are, in
// order, an ext, which is 0, the batch's segment_tokens and segment_bytes, and
// segment_hashes, where segment_hashes[i] is the SHA-256 of the bytes of xdr
// that segment_bytes[i] counts. The segments hash in parallel. cxs encodes
// each array behind its length and the segment hashes in order, so nothing is
// combined by XOR and no two valid batches share a preimage (hogd#4508).
//
// Go has no cxs encoder here (goxdr generates XDR only), so the cxs encoding
// of BatchCommit is written out, field by field. Its ext takes one byte.
//
// batch must be pre-validated with ValidateBatch. HashParallelBatch does not
// check the shape itself: bytes of xdr past the counted segments are left out
// of the hash, counts that overrun xdr are not reliably caught, and count
// arrays of unequal length are hashed as they are.
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
	preimage := make([]byte, 0, len(parallelBatchDomainSeparator)+1+5*(3+ncounts)+sha256.Size*len(segHashes))
	preimage = append(preimage, parallelBatchDomainSeparator...)
	// The ext of the BatchCommit: its only version is 0, one byte in cxs.
	preimage = append(preimage, 0)
	preimage = appendCxsUint32s(preimage, batch.Segment_tokens)
	preimage = appendCxsUint32s(preimage, batch.Segment_bytes)
	preimage = appendCxsUint32(preimage, uint32(len(segHashes)))

	wg.Wait()
	for i := range segHashes {
		preimage = append(preimage, segHashes[i][:]...)
	}
	return proto.SHA256Hash(sha256.Sum256(preimage))
}
