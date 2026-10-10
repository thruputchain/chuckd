package blocktrace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceID(t *testing.T) {
	var network [32]byte
	copy(network[:], bytes.Repeat([]byte{0x11}, 32))
	input := append([]byte("groundhog.block-trace.v1"), 0x00)
	input = append(input, bytes.Repeat([]byte{0x11}, 32)...)
	input = append(input, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2a)
	want := sha256.Sum256(input)
	id := TraceID(network, 42)
	if id != trace.TraceID(want[:16]) || id.String() != "553a3192399806a83a6f64004c4cab48" {
		t.Fatalf("unexpected trace ID: %s", id)
	}
}

func TestIDGenerator(t *testing.T) {
	p := sdktrace.NewTracerProvider(sdktrace.WithIDGenerator(IDGenerator()), sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() { _ = p.Shutdown(context.Background()) })
	tracer := p.Tracer("test")
	id := TraceID([32]byte{1}, 42)
	ctx, root := tracer.Start(WithTraceID(context.Background(), id), "root", trace.WithNewRoot())
	defer root.End()
	_, child := tracer.Start(ctx, "child")
	defer child.End()
	if root.SpanContext().TraceID() != id || child.SpanContext().TraceID() != id ||
		!root.SpanContext().SpanID().IsValid() || !child.SpanContext().SpanID().IsValid() ||
		root.SpanContext().SpanID() == child.SpanContext().SpanID() {
		t.Fatal("incorrect root or child IDs")
	}
	_, random := tracer.Start(context.Background(), "random")
	defer random.End()
	if !random.SpanContext().IsValid() || random.SpanContext().TraceID() == id {
		t.Fatal("incorrect random root IDs")
	}
}
