package benchxdr

import (
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func BenchmarkMarshalParallel(b *testing.B) {
	const n = 1e6
	x := proto.ParallelSignedTransactionHashBatch{
		Index: make([]byte, n*4),
		Xdr:   make([]byte, n*32),
	}
	m := libxdr.NewXdrMarshaler()
	for b.Loop() {
		bb := m.MarshalXdr(&x)
		bb.Recycle()
	}
}

func BenchmarkUnmarshalParallel(b *testing.B) {
	const n = 1e6
	x := proto.ParallelSignedTransactionHashBatch{
		Index: make([]byte, n*4),
		Xdr:   make([]byte, n*32),
	}
	m := libxdr.NewXdrMarshaler()
	bb := m.MarshalXdr(&x)
	for b.Loop() {
		libxdr.UnmarshalXdr(&x, bb.Bytes())
	}
}
