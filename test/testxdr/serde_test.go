package benchxdr

import (
	"testing"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func Benchmark_HashCV_1M(b *testing.B) {
	cv := &proto.ConsensusValue{
		Nomination: proto.Nomination{
			Transaction_hashes:      make([]proto.SignedTransactionHash, 1e6), // 1M
			Transaction_nominations: make([]proto.NominatedTransaction, 1e6),
		},
	}
	m := libxdr.NewXdrMarshaler()
	for b.Loop() {
		protoutil.HashCV(m, cv)
	}
}

func Benchmark_FastHashCV_1M(b *testing.B) {
	cv := &proto.ConsensusValue{
		Nomination: proto.Nomination{
			Transaction_hashes:      make([]proto.SignedTransactionHash, 1e6), // 1M
			Transaction_nominations: make([]proto.NominatedTransaction, 1e6),
		},
	}
	m := libxdr.NewXdrMarshaler()
	for b.Loop() {
		protoutil.FastHashCV(m, cv)
	}
}
