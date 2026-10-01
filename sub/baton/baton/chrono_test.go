package baton

import (
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestBlockSpanTree(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	old := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(old) })
	t0 := time.Unix(100, 0)
	x := blockChrono{height: 42, nominateStart: t0,
		mempoolNominateEnd: t0.Add(time.Second), nominateEnd: t0.Add(2 * time.Second),
		commitStart: t0.Add(3 * time.Second), commitEnd: t0.Add(4 * time.Second)}
	x.emitSpans()
	x.emitSpans()
	spans := recorder.Ended()
	if len(spans) != 4 {
		t.Fatalf("got %d spans, want 4", len(spans))
	}
	root := spans[3]
	if root.Name() != "block" || root.Parent().IsValid() || !root.StartTime().Equal(t0) || !root.EndTime().Equal(x.commitEnd) {
		t.Fatal("incorrect block root or timestamps")
	}
	for i, name := range []string{"propose", "nominate", "commit"} {
		if spans[i].Name() != name || spans[i].Parent().SpanID() != root.SpanContext().SpanID() || spans[i].SpanContext().TraceID() != root.SpanContext().TraceID() {
			t.Fatalf("incorrect child %s", name)
		}
	}
}
