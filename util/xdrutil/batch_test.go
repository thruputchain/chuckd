package xdrutil

import (
	crand "crypto/rand"
	"math/rand"
	"reflect"
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

const n = 1e6
const nseg = 30

func TestParallelizeSignedTxHashRoundtrip(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	in := make([]proto.SignedTransactionHash, 10)
	for i := range in {
		crand.Read(in[i][:])
	}
	x := ParallelizeSignedTransactionHashBatch(m, nseg, in)
	out, err := SerializeSignedTransactionHashBatch(x)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("not equal")
	}
}

func TestParallelizeSignedTxRoundtrip(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	in := make([]proto.SignedTransaction, 10)
	for i := range in {
		in[i].Tx.Uniquifier.Nonce = uint64(rand.Int63())
	}
	x := ParallelizeSignedTransactionBatch(m, nseg, in)
	out, err := SerializeSignedTransactionBatch(x)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != len(out) {
		t.Fatal("not equal length")
	}
	for i := range in {
		if in[i].Tx.Uniquifier.Nonce != out[i].Tx.Uniquifier.Nonce {
			t.Fatal("not equal")
		}
	}
}

func TestParallelizeNondeterministicProposalDataRoundtrip(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	in := make([]proto.NondeterministicProposalData, 10)
	for i := range in {
		in[i].Nondeterministic_bit_count = rand.Uint32()
	}
	x := ParallelizeNondeterministicProposalDataBatch(m, nseg, in)
	out, err := SerializeNondeterministicProposalDataBatch(x)
	if err != nil {
		t.Fatal(err)
	}
	if len(in) != len(out) {
		t.Fatal("not equal length")
	}
	for i := range in {
		if in[i].Nondeterministic_bit_count != out[i].Nondeterministic_bit_count {
			t.Fatal("not equal")
		}
	}
}

func BenchmarkParallelizeSignedTx(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransaction, n)
	for b.Loop() {
		ParallelizeSignedTransactionBatch(m, nseg, tokens)
	}
}

func BenchmarkParallelizeSignedTxHash(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransactionHash, n)
	for b.Loop() {
		ParallelizeSignedTransactionHashBatch(m, nseg, tokens)
	}
}

func BenchmarkParallelizeNondeterministicProposalData(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.NondeterministicProposalData, n)
	for b.Loop() {
		ParallelizeNondeterministicProposalDataBatch(m, nseg, tokens)
	}
}

func BenchmarkSerializeSignedTx(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransaction, n)
	x := ParallelizeSignedTransactionBatch(m, nseg, tokens)
	for b.Loop() {
		_, err := SerializeSignedTransactionBatch(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSerializeSignedTxHash(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransactionHash, n)
	x := ParallelizeSignedTransactionHashBatch(m, nseg, tokens)
	for b.Loop() {
		_, err := SerializeSignedTransactionHashBatch(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSerializeNondeterministicProposalData(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.NondeterministicProposalData, n)
	x := ParallelizeNondeterministicProposalDataBatch(m, nseg, tokens)
	for b.Loop() {
		_, err := SerializeNondeterministicProposalDataBatch(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}
