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
			0xe9, 0xd1, 0xec, 0xea, 0x10, 0x86, 0x05, 0xd0,
			0x61, 0xcb, 0x8d, 0xaf, 0xb3, 0x96, 0x2b, 0x43,
			0xc9, 0xf9, 0x66, 0xd0, 0xe0, 0xb5, 0x6b, 0xb7,
			0x6a, 0x4f, 0xb9, 0x0b, 0xb0, 0x02, 0xbf, 0x0d,
		}},
		{"multibyte", proto.ParallelBatch{
			Segment_tokens: []uint32{63, 300, 70000},
			Segment_bytes:  []uint32{1, 2, 3},
			Xdr:            []byte{1, 2, 3, 4, 5, 6},
		}, proto.SHA256Hash{
			0x51, 0xe3, 0x89, 0x59, 0x81, 0x61, 0x4d, 0x63,
			0x5e, 0x2c, 0xed, 0xc6, 0x9a, 0xae, 0x85, 0xc4,
			0xe1, 0x49, 0x06, 0x3d, 0x2b, 0x05, 0xb8, 0x95,
			0x7c, 0xb2, 0x51, 0x00, 0x84, 0xe8, 0x68, 0x90,
		}},
		{"many segments", manySegments, proto.SHA256Hash{
			0x9f, 0xd6, 0xa1, 0x93, 0x8f, 0x57, 0x79, 0x4d,
			0x14, 0x75, 0x83, 0x68, 0xf5, 0xa6, 0xe7, 0xbf,
			0xcc, 0x82, 0x90, 0x05, 0x22, 0x31, 0x89, 0x1d,
			0x20, 0xba, 0x9e, 0xc6, 0x90, 0xf3, 0x7a, 0x6d,
		}},
	} {
		if got := HashParallelBatch(tc.batch); got != tc.want {
			t.Errorf("%s batch hash: got %x, want %x", tc.name, got, tc.want)
		}
	}
}
