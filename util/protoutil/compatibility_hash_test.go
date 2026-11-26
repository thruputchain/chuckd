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
		0x55, 0x6a, 0xf4, 0x47, 0x8a, 0x82, 0xe4, 0x99,
		0x1f, 0x0e, 0x59, 0x42, 0xaf, 0xce, 0x8b, 0x66,
		0xe6, 0x83, 0x4c, 0xb4, 0xec, 0xf8, 0x82, 0x65,
		0xd8, 0xac, 0xa7, 0x48, 0x5d, 0x2d, 0x3e, 0xa3,
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
