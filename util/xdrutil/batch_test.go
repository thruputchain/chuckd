package xdrutil

import (
	"reflect"
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

const n = 1e6
const nseg = 30

func TestMarshalSignedTxRoundtrip(t *testing.T) {
	m := libxdr.NewXdrMarshaler()
	in := make([]proto.SignedTransaction, 3)
	x := MarshalSignedTransactionBatch(m, nseg, in)
	out, err := UnmarshalSignedTransactionBatch(x)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal("not equal")
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
