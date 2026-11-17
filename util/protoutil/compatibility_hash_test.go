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
		0x19, 0x15, 0x5c, 0x8d, 0x97, 0x54, 0x68, 0xd7,
		0xf3, 0x78, 0xf0, 0x07, 0xb2, 0x31, 0x80, 0xde,
		0x86, 0x0e, 0xf5, 0x4b, 0xae, 0xdb, 0x14, 0x9a,
		0xeb, 0x2d, 0x65, 0xb3, 0xbf, 0x6f, 0xb7, 0xb0,
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
