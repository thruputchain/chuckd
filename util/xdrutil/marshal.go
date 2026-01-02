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

func MarshalSignedTransactionBatch(
	m libxdr.XdrMarshaler,
	nSegment int,
	tokens []proto.SignedTransaction,
) proto.ParallelSignedTransactionBatch {
	return marshalBatch(m, nSegment, tokens,
		func(x *proto.SignedTransaction) xdr.XdrType {
			return proto.XDR_SignedTransaction(x)
		},
	)
}

func MarshalSignedTransactionHashBatch(
	m libxdr.XdrMarshaler,
	nGo int,
	tokens []proto.SignedTransactionHash,
) proto.ParallelSignedTransactionHashBatch {
	return marshalBatch(m, nGo, tokens,
		func(x *proto.SignedTransactionHash) xdr.XdrType {
			return proto.XDR_SignedTransactionHash(x)
		},
	)
}

func MarshalNondeterministicProposalDataBatch(
	m libxdr.XdrMarshaler,
	nGo int,
	tokens []proto.NondeterministicProposalData,
) proto.ParallelNondeterministicProposalDataBatch {
	return marshalBatch(m, nGo, tokens,
		func(x *proto.NondeterministicProposalData) xdr.XdrType {
			return proto.XDR_NondeterministicProposalData(x)
		},
	)
}

func marshalBatch[T any](
	m libxdr.XdrMarshaler,
	maxSegments int,
	tokens []T,
	toXDR func(*T) xdr.XdrType,
) proto.ParallelBatch {

	// determine number of sections
	nTokens := len(tokens)
	nTokensPerSegment := util.CeilDivSigned(nTokens, maxSegments) // tokens per segment = num tokens / num segments
	numSegments := util.CeilDivSigned(nTokens, nTokensPerSegment) // num useful segments = num tokens / num tokens per segment

	segmentNumTokens := make([]uint32, numSegments)
	segmentNumBytes := make([]uint32, numSegments)
	segmentBuf := make([]libmem.BorrowedBuffer, numSegments)
	var ntotal atomic.Uint32

	// marshal sections
	var wg sync.WaitGroup
	wg.Add(numSegments)
	for i := range numSegments {
		go func() {
			defer wg.Done()

			lo := i * nTokensPerSegment
			hi := min((i+1)*nTokensPerSegment, nTokens)
			segmentNumTokens[i] = uint32(hi - lo)

			segmentBuf[i] = m.Borrow()
			xout := xdr.XdrOut{Out: segmentBuf[i]}

			for _, token := range tokens[lo:hi] {
				xout.Marshal("", toXDR(&token))
			}

			k := uint32(segmentBuf[i].Len())
			segmentNumBytes[i] = k
			ntotal.Add(k)
		}()
	}
	wg.Wait()

	// merge sections
	xbuf := bytes.NewBuffer(make([]byte, 0, int(ntotal.Load())))
	for _, section := range segmentBuf {
		xbuf.Write(section.Bytes())
		section.Recycle()
	}

	return proto.ParallelBatch{
		Segment_tokens: segmentNumTokens,
		Segment_bytes:  segmentNumBytes,
		Xdr:            xbuf.Bytes(),
	}
}
