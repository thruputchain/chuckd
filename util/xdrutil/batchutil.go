package xdrutil

import (
	"fmt"

	proto "github.com/groundhogchain/protocol-go"
)

func ValidateBatch(batch proto.ParallelBatch) error {
	if len(batch.Segment_bytes) != len(batch.Segment_tokens) {
		return fmt.Errorf("number of byte and token segments mismatch")
	}
	nbytes := 0
	for _, k := range batch.Segment_bytes {
		nbytes += int(k)
	}
	if nbytes != len(batch.Xdr) {
		return fmt.Errorf("batch byte size mismatches index")
	}
	return nil
}

func TokenSizeOfBatch(x proto.ParallelBatch) int {
	n := 0
	for _, s := range x.Segment_tokens {
		n += int(s)
	}
	return n
}

func ByteSizeOfBatch(x proto.ParallelBatch) int {
	n := 0
	for _, s := range x.Segment_bytes {
		n += int(s)
	}
	return n
}

func TokenSizeOfBatches(x []proto.ParallelBatch) int {
	n := 0
	for _, b := range x {
		n += TokenSizeOfBatch(b)
	}
	return n
}

func JoinBatches(bs []proto.ParallelBatch) proto.ParallelBatch {
	r := proto.ParallelBatch{}
	for _, b := range bs {
		r.Segment_tokens = append(r.Segment_tokens, b.Segment_tokens...)
		r.Segment_bytes = append(r.Segment_bytes, b.Segment_bytes...)
		r.Xdr = append(r.Xdr, b.Xdr...)
	}
	return r
}
