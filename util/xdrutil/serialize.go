package xdrutil

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

func SerializeSignedTransactionBatch(batch proto.ParallelBatch) ([]proto.SignedTransaction, error) {
	return serializeBatch(batch,
		func(x *proto.SignedTransaction) xdr.XdrType {
			return proto.XDR_SignedTransaction(x)
		},
	)
}

func SerializeSignedTransactionHashBatch(batch proto.ParallelBatch) ([]proto.SignedTransactionHash, error) {
	return serializeBatch(batch,
		func(x *proto.SignedTransactionHash) xdr.XdrType {
			return proto.XDR_SignedTransactionHash(x)
		},
	)
}

func SerializeNondeterministicProposalDataBatch(batch proto.ParallelBatch) ([]proto.NondeterministicProposalData, error) {
	return serializeBatch(batch,
		func(x *proto.NondeterministicProposalData) xdr.XdrType {
			return proto.XDR_NondeterministicProposalData(x)
		},
	)
}

func serializeBatch[T any](batch proto.ParallelBatch, toXDR func(*T) xdr.XdrType) (tokens []T, _ error) {

	t0 := time.Now()
	defer func() {
		liblog.Debugf("elapsed %v to serialize batch with %d tokens from %d segments", time.Since(t0), len(tokens), len(batch.Segment_tokens))
	}()

	if err := ValidateBatch(batch); err != nil {
		return nil, err
	}
	numSegments := len(batch.Segment_bytes)

	// determine total number of tokens
	numTokens := 0
	for _, nSegmentTokens := range batch.Segment_tokens {
		numTokens += int(nSegmentTokens)
	}
	if numTokens == 0 {
		return []T{}, nil
	}
	tokens = make([]T, numTokens)

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
