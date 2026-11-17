package xdrutil

import (
	"testing"

	proto "github.com/thruputchain/thruput-protocol-go"
)

// With segment hashes combined by XOR, swapping two segments of the same size
// and token count kept the batch hash, so one hash named two batches
// (hogd#4508).
func TestHashParallelBatchBindsSegmentOrder(t *testing.T) {
	first := proto.SignedTransactionHash{1}
	second := proto.SignedTransactionHash{2}
	batch := proto.ParallelBatch{
		Segment_tokens: []uint32{1, 1},
		Segment_bytes:  []uint32{32, 32},
		Xdr:            append(first[:], second[:]...),
	}
	swapped := batch
	swapped.Xdr = append(second[:], first[:]...)

	if HashParallelBatch(batch) == HashParallelBatch(swapped) {
		t.Fatal("swapping two segments kept the batch hash")
	}
}

// The token counts are part of the preimage, though no segment hash reads them.
func TestHashParallelBatchBindsSegmentTokens(t *testing.T) {
	batch := proto.ParallelBatch{
		Segment_tokens: []uint32{2, 2},
		Segment_bytes:  []uint32{8, 8},
		Xdr:            []byte{0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 4},
	}
	recounted := batch
	recounted.Segment_tokens = []uint32{1, 3}

	if HashParallelBatch(batch) == HashParallelBatch(recounted) {
		t.Fatal("changing the token counts kept the batch hash")
	}
}

// Shared with the coordinator's hash_parallel_batch_golden_vectors and hogd's
// Mempool.ParallelBatchHashGoldenVectors. The second batch has token counts in
// the 2, 3 and 5 byte cxs forms. The third has 64 segments, so the three array
// lengths take 2 bytes, and multi-byte byte counts.
func TestHashParallelBatchVectors(t *testing.T) {
	manySegments := proto.ParallelBatch{
		Segment_tokens: make([]uint32, 64),
		Segment_bytes:  make([]uint32, 64),
		Xdr:            make([]byte, 70362),
	}
	for i := range manySegments.Segment_tokens {
		manySegments.Segment_tokens[i] = uint32(i)
		manySegments.Segment_bytes[i] = 1
	}
	manySegments.Segment_bytes[0] = 300
	manySegments.Segment_bytes[1] = 70000
	for j := range manySegments.Xdr {
		manySegments.Xdr[j] = byte(j % 251)
	}

	for _, tc := range []struct {
		name  string
		batch proto.ParallelBatch
		want  proto.SHA256Hash
	}{
		{"empty", proto.ParallelBatch{}, proto.SHA256Hash{
			0xcc, 0xdf, 0xcb, 0x6b, 0x0d, 0x5c, 0xbe, 0x32,
			0xf1, 0x2f, 0x44, 0x31, 0x2f, 0xf4, 0xee, 0xf3,
			0x73, 0xf1, 0x0c, 0x77, 0xca, 0xd5, 0x3f, 0xac,
			0x21, 0xed, 0x79, 0xed, 0x2f, 0xb6, 0xcc, 0x41,
		}},
		{"multibyte", proto.ParallelBatch{
			Segment_tokens: []uint32{63, 300, 70000},
			Segment_bytes:  []uint32{1, 2, 3},
			Xdr:            []byte{1, 2, 3, 4, 5, 6},
		}, proto.SHA256Hash{
			0x12, 0x51, 0x4d, 0x52, 0x2c, 0xac, 0xa7, 0xc9,
			0x3d, 0x29, 0xbd, 0xbe, 0x05, 0x73, 0x42, 0xd6,
			0x95, 0x60, 0x91, 0xb4, 0x74, 0xe5, 0x78, 0x48,
			0x6d, 0x56, 0x4b, 0x5a, 0x29, 0x00, 0x20, 0xdc,
		}},
		{"many segments", manySegments, proto.SHA256Hash{
			0x23, 0x75, 0x23, 0x22, 0xbf, 0xa2, 0x14, 0x52,
			0x37, 0x8b, 0x97, 0x1f, 0x7c, 0x88, 0xc7, 0xc4,
			0x5b, 0x41, 0x86, 0x91, 0x3b, 0x31, 0x9a, 0x8a,
			0xda, 0xcc, 0x3a, 0x4d, 0xf9, 0xc0, 0xe6, 0x12,
		}},
	} {
		if got := HashParallelBatch(tc.batch); got != tc.want {
			t.Errorf("%s batch hash: got %x, want %x", tc.name, got, tc.want)
		}
	}
}
