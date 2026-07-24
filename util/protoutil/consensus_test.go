package protoutil

import (
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestHashProposalTxsFastIsCanonicalXDRHash(t *testing.T) {
	txs := proto.ProposalTxs{
		Hash: proto.ParallelBatch{
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

	got := HashProposalTxsFast(&txs)
	want := HashProposalTxsSlow(&txs)
	if got != want {
		t.Fatalf("non-canonical ProposalTxs hash: got %x, want %x", got, want)
	}
}
