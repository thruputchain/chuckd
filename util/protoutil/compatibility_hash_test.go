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
		0xac, 0x7e, 0x6e, 0x65, 0xe3, 0xf3, 0x6e, 0xc7,
		0x88, 0x45, 0xdd, 0xf3, 0xf3, 0xd9, 0x5a, 0xcf,
		0x99, 0xea, 0xdb, 0xbd, 0xbc, 0xfd, 0x39, 0x1e,
		0x44, 0x23, 0x9a, 0x7d, 0xc4, 0xb5, 0x2c, 0x03,
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
