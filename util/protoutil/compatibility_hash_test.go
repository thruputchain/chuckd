package protoutil

import (
	"testing"

	proto "github.com/thruputchain/thruput-protocol-go"
)

func TestHashProposalTxsFastCompatibilityVector(t *testing.T) {
	txs := proto.ProposalTxs{
		Signed_hash: proto.ParallelBatch{
			Segment_tokens: []uint32{1, 2},
			Segment_bytes:  []uint32{4, 8},
			Xdr:            []byte{0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3},
		},
		Nondeterministic_data: proto.ParallelBatch{
			Segment_tokens: []uint32{1},
			Segment_bytes:  []uint32{4},
			Xdr:            []byte{0xde, 0xad, 0xbe, 0xef},
		},
	}

	// Shared with hogd's Mempool.ProposalTxsCompatibilityHashGoldenVector and
	// the coordinator's proposal_txs_hash_golden_vector.
	got := HashProposalTxsFast(&txs)
	want := proto.ProposalTxsHash{
		0x43, 0xf1, 0xc2, 0x9a, 0xad, 0x59, 0x64, 0x59,
		0x48, 0x1d, 0x80, 0x7f, 0x00, 0x8e, 0xbe, 0x2e,
		0x29, 0x77, 0xea, 0x0e, 0xd6, 0x4a, 0x3a, 0xdd,
		0x24, 0xa8, 0x95, 0xf1, 0x89, 0xa4, 0xde, 0xc6,
	}
	if got != want {
		t.Fatalf("compatibility ProposalTxs hash: got %x, want %x", got, want)
	}
	if got == HashProposalTxsSlow(&txs) {
		t.Fatal("compatibility hash unexpectedly equals canonical XDR hash")
	}
}

// With the batch hashes combined by XOR, two batches of the same shape could
// trade places and keep the ProposalTxs hash (hogd#4508).
func TestHashProposalTxsFastBindsBatchPositions(t *testing.T) {
	batch := func(word byte) proto.ParallelBatch {
		return proto.ParallelBatch{
			Segment_tokens: []uint32{1},
			Segment_bytes:  []uint32{4},
			Xdr:            []byte{0, 0, 0, word},
		}
	}
	txs := proto.ProposalTxs{
		Signed_hash:           batch(1),
		Nondeterministic_data: batch(2),
	}
	swapped := proto.ProposalTxs{
		Signed_hash:           batch(2),
		Nondeterministic_data: batch(1),
	}

	if HashProposalTxsFast(&txs) == HashProposalTxsFast(&swapped) {
		t.Fatal("swapping the two batches kept the ProposalTxs hash")
	}
}
