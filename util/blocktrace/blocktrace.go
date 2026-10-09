package blocktrace

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TraceID returns the deterministic trace ID of a block's trace.
func TraceID(networkHash [32]byte, height uint64) trace.TraceID {
	b := append([]byte("groundhog.block-trace.v1\x00"), networkHash[:]...)
	b = binary.BigEndian.AppendUint64(b, height)
	hash := sha256.Sum256(b)
	id := trace.TraceID(hash[:16])
	if !id.IsValid() {
		id[15] = 1
	}
	return id
}

type traceIDKey struct{}

// WithTraceID makes every root span started from ctx (or a descendant without
// an active span) use id as its trace ID when the tracer provider uses
// IDGenerator. Use it only to start the block root; do not pass ctx on.
func WithTraceID(ctx context.Context, id trace.TraceID) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}

type idGenerator struct{}

// IDGenerator returns an sdktrace.IDGenerator that uses the trace ID set by
// WithTraceID for new roots, and random IDs otherwise (span IDs are always random).
func IDGenerator() sdktrace.IDGenerator { return idGenerator{} }

func (g idGenerator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
	id, _ := ctx.Value(traceIDKey{}).(trace.TraceID)
	for !id.IsValid() {
		if _, err := rand.Read(id[:]); err != nil {
			panic(err)
		}
	}
	return id, g.NewSpanID(ctx, id)
}

func (idGenerator) NewSpanID(context.Context, trace.TraceID) trace.SpanID {
	var id trace.SpanID
	for !id.IsValid() {
		if _, err := rand.Read(id[:]); err != nil {
			panic(err)
		}
	}
	return id
}
