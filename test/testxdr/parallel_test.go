package benchxdr

import (
	"testing"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func BenchmarkMarshalParallel(b *testing.B) {
	const n = 1e6
	x := proto.ParallelTransactionHashBatch{
		Segment_bytes: make([]uint32, n*4),
		Xdr:           make([]byte, n*32),
	}
	m := libxdr.NewXdrMarshaler()
	for b.Loop() {
		bb := m.MarshalXdr(&x)
		bb.Recycle()
	}
}

func BenchmarkUnmarshalParallel(b *testing.B) {
	const n = 1e6
	x := proto.ParallelTransactionHashBatch{
		Segment_bytes: make([]uint32, n*4),
		Xdr:           make([]byte, n*32),
	}
	m := libxdr.NewXdrMarshaler()
	bb := m.MarshalXdr(&x)
	for b.Loop() {
		libxdr.UnmarshalXdr(&x, bb.Bytes())
	}
}
