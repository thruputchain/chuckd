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

func TestMarshalSignedTxHashRoundtrip(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	in := make([]proto.SignedTransactionHash, 10)
	for i := range in {
		crand.Read(in[i][:])
	}
	x := MarshalSignedTransactionHashBatch(m, nseg, in)
	out, err := UnmarshalSignedTransactionHashBatch(x)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("not equal")
	}
}

func TestMarshalSignedTxRoundtrip(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	in := make([]proto.SignedTransaction, 10)
	for i := range in {
		in[i].Tx.Uniquifier.Nonce = uint64(rand.Int63())
	}
	x := MarshalSignedTransactionBatch(m, nseg, in)
	out, err := UnmarshalSignedTransactionBatch(x)
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

func TestMarshalNondeterministicProposalDataRoundtrip(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	in := make([]proto.NondeterministicProposalData, 10)
	for i := range in {
		in[i].Nondeterministic_bit_count = rand.Uint32()
	}
	x := MarshalNondeterministicProposalDataBatch(m, nseg, in)
	out, err := UnmarshalNondeterministicProposalDataBatch(x)
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

func BenchmarkMarshalSignedTx(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransaction, n)
	for b.Loop() {
		MarshalSignedTransactionBatch(m, nseg, tokens)
	}
}

func BenchmarkMarshalSignedTxHash(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransactionHash, n)
	for b.Loop() {
		MarshalSignedTransactionHashBatch(m, nseg, tokens)
	}
}

func BenchmarkMarshalNondeterministicProposalData(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.NondeterministicProposalData, n)
	for b.Loop() {
		MarshalNondeterministicProposalDataBatch(m, nseg, tokens)
	}
}

func BenchmarkUnmarshalSignedTx(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransaction, n)
	x := MarshalSignedTransactionBatch(m, nseg, tokens)
	for b.Loop() {
		_, err := UnmarshalSignedTransactionBatch(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalSignedTxHash(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransactionHash, n)
	x := MarshalSignedTransactionHashBatch(m, nseg, tokens)
	for b.Loop() {
		_, err := UnmarshalSignedTransactionHashBatch(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalNondeterministicProposalData(b *testing.B) {
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.NondeterministicProposalData, n)
	x := MarshalNondeterministicProposalDataBatch(m, nseg, tokens)
	for b.Loop() {
		_, err := UnmarshalNondeterministicProposalDataBatch(x)
		if err != nil {
			b.Fatal(err)
		}
	}
}
