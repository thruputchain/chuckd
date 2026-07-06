package benchxdr

import (
	"bytes"
	"crypto/sha256"
	"io"
	"testing"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func mkProposal() *proto.Proposal {
	txh := make([]proto.SignedTransactionHash, 1e6) // 1M
	txn := make([]proto.NondeterministicProposalData, 1e6)
	m := libxdr.NewXdrMarshaler()
	return &proto.Proposal{
		Transactions: proto.ProposalTxs{
			Hash: xdrutil.ParallelizeSignedTransactionHashBatch(m, 20, txh),
			Nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(m, 20, txn),
		},
	}
}

func Benchmark_HashProposalSlow_1M(b *testing.B) {
	proposal := mkProposal()
	for b.Loop() {
		protoutil.HashProposalSlow(proposal)
	}
}

func Benchmark_HashProposalFast_1M(b *testing.B) {
	proposal := mkProposal()
	for b.Loop() {
		protoutil.HashProposalFast(proposal)
	}
}

func Benchmark_Marshal_Nop_1M(b *testing.B) {
	proposal := mkProposal()
	for b.Loop() {
		libxdr.MarshalXdrWriter(io.Discard, proposal)
	}
}

func Benchmark_Marshal_NoAlloc_1M(b *testing.B) {
	proposal := mkProposal()
	buf := make([]byte, 1e6*100)
	w := bytes.NewBuffer(buf)
	for b.Loop() {
		libxdr.MarshalXdrWriter(w, proposal)
		w.Reset()
	}
}

func Benchmark_Marshal_PoolAlloc_1M(b *testing.B) {
	proposal := mkProposal()
	m := libxdr.NewXdrMarshaler()
	for b.Loop() {
		b := m.MarshalXdr(proposal)
		b.Recycle()
	}
}

func Benchmark_Marshal_Sha256_1M(b *testing.B) {
	proposal := mkProposal()
	for b.Loop() {
		libxdr.MarshalXdrWriter(sha256.New(), proposal)
	}
}
