package xdrutil

import (
	"bytes"
	"context"
	"fmt"
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
	nSegment int,
	tokens []T,
	toXDR func(*T) xdr.XdrType,
) proto.ParallelBatch {

	// determine number of sections
	nTokens := len(tokens)
	nTokensPerSegment := util.CeilDivSigned(nTokens, nSegment)  // tokens per segment = num tokens / num segments
	nSegments := util.CeilDivSigned(nTokens, nTokensPerSegment) // num useful segments = num tokens / num tokens per segment

	segmentNumTokens := make([]uint32, nSegments)
	segmentNumBytes := make([]uint32, nSegments)
	segmentBuf := make([]libmem.BorrowedBuffer, nSegments)
	var ntotal atomic.Uint32

	// marshal sections
	var wg sync.WaitGroup
	wg.Add(nSegments)
	for i := range nSegments {
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

func UnmarshalSignedTransactionBatch(batch proto.ParallelBatch) ([]proto.SignedTransaction, error) {
	return unmarshalBatch(batch,
		func(x *proto.SignedTransaction) xdr.XdrType {
			return proto.XDR_SignedTransaction(x)
		},
	)
}

func UnmarshalSignedTransactionHashBatch(batch proto.ParallelBatch) ([]proto.SignedTransactionHash, error) {
	return unmarshalBatch(batch,
		func(x *proto.SignedTransactionHash) xdr.XdrType {
			return proto.XDR_SignedTransactionHash(x)
		},
	)
}

func UnmarshalNondeterministicProposalDataBatch(batch proto.ParallelBatch) ([]proto.NondeterministicProposalData, error) {
	return unmarshalBatch(batch,
		func(x *proto.NondeterministicProposalData) xdr.XdrType {
			return proto.XDR_NondeterministicProposalData(x)
		},
	)
}

func unmarshalBatch[T any](batch proto.ParallelBatch, toXDR func(*T) xdr.XdrType) ([]T, error) {

	// XXX: validate
	if len(batch.Segment_bytes) != len(batch.Segment_tokens) {
		return nil, fmt.Errorf("number of byte and token segments mismatch")
	}
	numSegments := len(batch.Segment_bytes)

	// determine total number of tokens
	numTokens := 0
	for _, nSegmentTokens := range batch.Segment_tokens {
		numTokens += int(nSegmentTokens)
	}
	tokens := make([]T, numTokens)

	// unmarshal segments
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	var wg sync.WaitGroup
	wg.Add(numSegments)

	offsetTokens := 0
	offsetBytes := 0
	for i := range numSegments {
		segmentNumTokens := int(batch.Segment_tokens[i])
		segmentNumBytes := int(batch.Segment_bytes[i])

		go func(offsetTokens int, offsetBytes int) {
			defer wg.Done()

			buf := bytes.NewBuffer(batch.Xdr[offsetBytes : offsetBytes+segmentNumBytes])

			for j := range segmentNumTokens {
				if ctx.Err() != nil {
					return // context cancelled
				}
				err := libxdr.UnmarshalXdrBuffer(toXDR(&tokens[offsetTokens+j]), buf)
				if err != nil {
					cancel(err) // abort all goroutines
					return
				}
			}

		}(offsetTokens, offsetBytes)

		offsetTokens += segmentNumTokens
		offsetBytes += segmentNumBytes
	}
	wg.Wait()

	return tokens, ctx.Err()
}
