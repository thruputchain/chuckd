package xdrutil

import (
	"bytes"
	"sync"
	"sync/atomic"

	"github.com/groundhogchain/chuckd/util"
	"github.com/groundhogchain/lib-go/libmem"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

func MarshalParallelSignedTransaction(
	sectionM libxdr.XdrMarshaler,
	nGo int,
	tokens []proto.SignedTransaction,
) proto.ParallelSignedTransactionBatch {
	index, xdr := marshalParallel(sectionM, nGo, tokens,
		func(x *proto.SignedTransaction) xdr.XdrType {
			return proto.XDR_SignedTransaction(x)
		},
	)
	return proto.ParallelSignedTransactionBatch{Index: index, Xdr: xdr}
}

func MarshalParallelSignedTransactionHash(
	sectionM libxdr.XdrMarshaler,
	nGo int,
	tokens []proto.SignedTransactionHash,
) proto.ParallelSignedTransactionHashBatch {
	index, xdr := marshalParallel(sectionM, nGo, tokens,
		func(x *proto.SignedTransactionHash) xdr.XdrType {
			return proto.XDR_SignedTransactionHash(x)
		},
	)
	return proto.ParallelSignedTransactionHashBatch{Index: index, Xdr: xdr}
}

func MarshalParallelNondeterministicProposalData(
	sectionM libxdr.XdrMarshaler,
	nGo int,
	tokens []proto.NondeterministicProposalData,
) proto.ParallelNondeterministicProposalDataBatch {
	index, xdr := marshalParallel(sectionM, nGo, tokens,
		func(x *proto.NondeterministicProposalData) xdr.XdrType {
			return proto.XDR_NondeterministicProposalData(x)
		},
	)
	return proto.ParallelNondeterministicProposalDataBatch{Index: index, Xdr: xdr}
}

func marshalParallel[T any](
	sectionM libxdr.XdrMarshaler,
	nGo int,
	tokens []T,
	toXDR func(*T) xdr.XdrType, // proto.XDR_SignedTransactionHash
) ([]uint32, []byte) {

	// determine number of sections
	nTokens := len(tokens)
	nTokensPerGo := util.CeilDivSigned(nTokens, nGo)       // tokens per goroutine = num tokens / num goroutines
	nSections := util.CeilDivSigned(nTokens, nTokensPerGo) // num useful goroutines = num tokens / num tokens per goroutine

	index := make([]uint32, nSections)
	sections := make([]libmem.BorrowedBuffer, nSections)
	var ntotal atomic.Uint32

	// marshal sections
	var wg sync.WaitGroup
	wg.Add(nSections)
	for i := range nSections {
		go func() {
			defer wg.Done()

			lo := i * nTokensPerGo
			hi := min((i+1)*nTokensPerGo, nTokens)

			sections[i] = sectionM.Borrow()
			xout := xdr.XdrOut{Out: sections[i]}

			for _, token := range tokens[lo:hi] {
				xout.Marshal("", toXDR(&token))
			}

			k := uint32(sections[i].Len())
			index[i] = k
			ntotal.Add(k)
		}()
	}
	wg.Wait()

	// merge sections
	xbuf := bytes.NewBuffer(make([]byte, 0, int(ntotal.Load())))
	for _, section := range sections {
		xbuf.Write(section.Bytes())
		section.Recycle()
	}

	return index, xbuf.Bytes()
}
