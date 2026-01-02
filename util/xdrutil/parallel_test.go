package xdrutil

import (
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func BenchmarkMarshalParallelSignedTx(b *testing.B) {
	const n = 1e6
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransaction, n)
	for b.Loop() {
		MarshalParallelSignedTransaction(m, 30, tokens)
	}
}

func BenchmarkMarshalParallelSignedTxHash(b *testing.B) {
	const n = 1e6
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransactionHash, n)
	for b.Loop() {
		MarshalParallelSignedTransactionHash(m, 30, tokens)
	}
}

func BenchmarkMarshalNondeterministicProposalData(b *testing.B) {
	const n = 1e6
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.NondeterministicProposalData, n)
	for b.Loop() {
		MarshalParallelNondeterministicProposalData(m, 30, tokens)
	}
}
