package xdrutil

import (
	"testing"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func BenchmarkMarshalParallel(b *testing.B) {
	const n = 1e6
	m := libxdr.NewXdrMarshaler()
	tokens := make([]proto.SignedTransactionHash, n)
	for b.Loop() {
		marshalParallel(m, 20, tokens)
	}
}
