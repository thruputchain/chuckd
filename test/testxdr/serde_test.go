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

func mkCV() *proto.ConsensusValue {
	txh := make([]proto.SignedTransactionHash, 1e6) // 1M
	txn := make([]proto.NondeterministicProposalData, 1e6)
	m := libxdr.NewXdrMarshaler()
	return &proto.ConsensusValue{
		Nomination: proto.Nomination{
			Transactions_hash:                  xdrutil.ParallelizeSignedTransactionHashBatch(m, 20, txh),
			Transactions_nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(m, 20, txn),
		},
	}
}

func Benchmark_HashCV_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	cv := mkCV()
	for b.Loop() {
		protoutil.HashCV(cv)
	}
}

func Benchmark_Marshal_Nop_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	cv := mkCV()
	for b.Loop() {
		libxdr.MarshalXdrWriter(io.Discard, cv)
	}
}

func Benchmark_Marshal_NoAlloc_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	cv := mkCV()
	buf := make([]byte, 1e6*100)
	w := bytes.NewBuffer(buf)
	for b.Loop() {
		libxdr.MarshalXdrWriter(w, cv)
		w.Reset()
	}
}

func Benchmark_Marshal_PoolAlloc_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	cv := mkCV()
	m := libxdr.NewXdrMarshaler()
	for b.Loop() {
		b := m.MarshalXdr(cv)
		b.Recycle()
	}
}

func Benchmark_Marshal_Sha256_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	cv := mkCV()
	for b.Loop() {
		libxdr.MarshalXdrWriter(sha256.New(), cv)
	}
}
