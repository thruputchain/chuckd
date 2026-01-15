package benchxdr

import (
	"testing"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func Benchmark_HashCV_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	txh := make([]proto.SignedTransactionHash, 1e6) // 1M
	txn := make([]proto.NondeterministicProposalData, 1e6)
	m := libxdr.NewXdrMarshaler()
	cv := &proto.ConsensusValue{
		Nomination: proto.Nomination{
			Transactions_hash:                  xdrutil.ParallelizeSignedTransactionHashBatch(m, 20, txh),
			Transactions_nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(m, 20, txn),
		},
	}
	for b.Loop() {
		protoutil.HashCV(m, cv)
	}
}
