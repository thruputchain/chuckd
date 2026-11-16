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
// the 2, 3 and 5 byte cxs forms. The third has 64 segments, so both array
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
			0x9b, 0x1a, 0xe5, 0xf1, 0xd0, 0xee, 0xc3, 0xff,
			0x77, 0x06, 0xd9, 0xf6, 0x6e, 0x0e, 0x39, 0x60,
			0xb3, 0x94, 0xba, 0xe6, 0xdf, 0x89, 0xda, 0xa7,
			0x42, 0x19, 0xbd, 0x75, 0x69, 0x9e, 0xed, 0x84,
		}},
		{"multibyte", proto.ParallelBatch{
			Segment_tokens: []uint32{63, 300, 70000},
			Segment_bytes:  []uint32{1, 2, 3},
			Xdr:            []byte{1, 2, 3, 4, 5, 6},
		}, proto.SHA256Hash{
			0x04, 0xd5, 0x83, 0x41, 0xd9, 0xc4, 0xad, 0x95,
			0x38, 0xd7, 0xdc, 0x7d, 0x0b, 0x92, 0x0e, 0xd4,
			0x0c, 0xb9, 0x00, 0xa9, 0x26, 0x41, 0xcf, 0xd6,
			0x2a, 0x61, 0x42, 0x55, 0xd0, 0x12, 0x25, 0xfa,
		}},
		{"many segments", manySegments, proto.SHA256Hash{
			0x49, 0x7d, 0xf8, 0xdb, 0x36, 0x4d, 0xd1, 0xbf,
			0x0f, 0xf5, 0x0b, 0xf0, 0xa8, 0x25, 0xab, 0x57,
			0xd7, 0xa6, 0x55, 0xb4, 0xf9, 0xc5, 0xde, 0x64,
			0xf7, 0x8d, 0xe9, 0x53, 0x91, 0x81, 0x75, 0xb2,
		}},
	} {
		if got := HashParallelBatch(tc.batch); got != tc.want {
			t.Errorf("%s batch hash: got %x, want %x", tc.name, got, tc.want)
		}
	}
}
