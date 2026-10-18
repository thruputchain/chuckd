package protoutil

import (
	"testing"

	proto "github.com/thruputchain/thruput-protocol-go"
)

func TestHashProposalTxsFastCompatibilityVector(t *testing.T) {
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
	want := proto.ProposalTxsHash{
		0x3a, 0x3c, 0xe7, 0x82, 0x83, 0x94, 0x38, 0x26,
		0xac, 0x43, 0x29, 0x1f, 0xab, 0x79, 0xbd, 0x17,
		0xc4, 0xad, 0x1d, 0x1e, 0xf9, 0xe2, 0x2d, 0x8c,
		0x72, 0x31, 0xa9, 0x7c, 0xb6, 0x68, 0xdc, 0xd9,
	}
	if got != want {
		t.Fatalf("compatibility ProposalTxs hash: got %x, want %x", got, want)
	}
	if got == HashProposalTxsSlow(&txs) {
		t.Fatal("compatibility hash unexpectedly equals canonical XDR hash")
	}
}
