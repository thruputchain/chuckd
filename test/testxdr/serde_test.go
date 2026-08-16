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

func mkBasis() *proto.ConsensusBasis {
	txh := make([]proto.TransactionHash, 1e6) // 1M
	txn := make([]proto.NondeterministicProposalData, 1e6)
	m := libxdr.NewXdrMarshaler()
	return &proto.ConsensusBasis{
		Proposal_txs: proto.ProposalTxs{
			Hash:                  xdrutil.ParallelizeTransactionHashBatch(m, 20, txh),
			Nondeterministic_data: xdrutil.ParallelizeNondeterministicProposalDataBatch(m, 20, txn),
		},
	}
}

func Benchmark_HashCVSlow_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	basis := mkBasis()
	for b.Loop() {
		protoutil.HashProposalTxsSlow(&basis.Proposal_txs)
	}
}

func Benchmark_HashCVFast_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	basis := mkBasis()
	for b.Loop() {
		protoutil.HashProposalTxsFast(&basis.Proposal_txs)
	}
}

func Benchmark_Marshal_Nop_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	basis := mkBasis()
	for b.Loop() {
		libxdr.MarshalXdrWriter(io.Discard, basis)
	}
}

func Benchmark_Marshal_NoAlloc_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	basis := mkBasis()
	buf := make([]byte, 1e6*100)
	w := bytes.NewBuffer(buf)
	for b.Loop() {
		libxdr.MarshalXdrWriter(w, basis)
		w.Reset()
	}
}

func Benchmark_Marshal_PoolAlloc_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	basis := mkBasis()
	m := libxdr.NewXdrMarshaler()
	for b.Loop() {
		b := m.MarshalXdr(basis)
		b.Recycle()
	}
}

func Benchmark_Marshal_Sha256_1M(b *testing.B) {
	// liblog.SetLogLevelDebug()
	basis := mkBasis()
	for b.Loop() {
		libxdr.MarshalXdrWriter(sha256.New(), basis)
	}
}
