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
		0x7f, 0xab, 0x78, 0x9c, 0x4a, 0xd1, 0x98, 0x7c,
		0x1c, 0x85, 0xd1, 0xdc, 0x39, 0x7f, 0xf9, 0x58,
		0x6a, 0x74, 0x7f, 0xeb, 0xee, 0x67, 0xbc, 0x4b,
		0x07, 0x02, 0xde, 0x82, 0x84, 0x1e, 0xb3, 0x3a,
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
