package xdrutil

import (
	"crypto/sha256"
	"sync"

	proto "github.com/thruputchain/thruput-protocol-go"
)

// parallelBatchDomainSeparator is cxs(DomainSeparator("ParallelBatch")), the
// first field of a BatchCommit: the zero ExtensionPoint, the length of the
// name, and the name.
var parallelBatchDomainSeparator = []byte("\x00\x0dParallelBatch")

// HashParallelBatch computes the hash that commits to a batch:
//
//	SHA-256(cxs(BatchCommit))
//
// BatchCommit is the struct of the protocol's parallel.x. Its fields are, in
// order, DomainSeparator("ParallelBatch"), the batch's segment_tokens and
// segment_bytes, and segment_hashes, where segment_hashes[i] is the SHA-256
// of the bytes of xdr that segment_bytes[i] counts. The segments hash in
// parallel. cxs encodes each array behind its length and the segment hashes
// in order, so nothing is combined by XOR and no two valid batches share a
// preimage (hogd#4508).
//
// The protocol revision chuckd builds with does not define BatchCommit yet,
// so its cxs encoding is written out here, field by field.
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
	commit := make([]byte, 0, len(parallelBatchDomainSeparator)+5*(3+ncounts)+sha256.Size*len(segHashes))
	commit = append(commit, parallelBatchDomainSeparator...)
	commit = appendCxsUint32s(commit, batch.Segment_tokens)
	commit = appendCxsUint32s(commit, batch.Segment_bytes)
	commit = appendCxsUint32(commit, uint32(len(segHashes)))

	wg.Wait()
	for i := range segHashes {
		commit = append(commit, segHashes[i][:]...)
	}
	return proto.SHA256Hash(sha256.Sum256(commit))
}
