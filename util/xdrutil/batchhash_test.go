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
