package baton

import (
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

func TestCommitSpanSurvivesPendingNomination(t *testing.T) {
	for _, prepared := range []bool{false, true} {
		t.Run(map[bool]string{false: "clock_skew_wait", true: "nomination_pending"}[prepared], func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			old := otel.GetTracerProvider()
			otel.SetTracerProvider(provider)
			t.Cleanup(func() { otel.SetTracerProvider(old) })
			t0 := time.Unix(100, 0)
			var c chrono
			x := c.at(42)
			x.nominateStart = t0
			if prepared {
				x.mempoolNominateEnd = t0.Add(time.Second)
			}
			x.commitStart = t0.Add(2 * time.Second)
			x.commitEnd = t0.Add(3 * time.Second)
			x.emitSpans()
			c.at(44) // Reuse the slot before the old proposal returns.
			spans := recorder.Ended()
			want := 2
			if prepared {
				want++
			}
			if len(spans) != want {
				t.Fatalf("got %d spans, want %d", len(spans), want)
			}
			root := spans[len(spans)-1]
			commit := spans[len(spans)-2]
			if root.Name() != "block" || commit.Name() != "commit" || commit.Parent().SpanID() != root.SpanContext().SpanID() || !root.EndTime().Equal(t0.Add(3*time.Second)) {
				t.Fatal("completed commit trace lost or incorrect")
			}
		})
	}
}

func TestLosingProposalSpanIdentity(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	old := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() { otel.SetTracerProvider(old) })
	t0 := time.Unix(100, 0)
	x := blockChrono{height: 42, nominateStart: t0,
		mempoolNominateEnd: t0.Add(time.Second), nominateEnd: t0.Add(2 * time.Second),
		commitStart: t0.Add(3 * time.Second), commitEnd: t0.Add(4 * time.Second),
		attributes: []attribute.KeyValue{attribute.String("value.hash_prefix", "winner")},
		proposalAttributes: []attribute.KeyValue{
			attribute.String("proposal.value.hash_prefix", "loser"),
			attribute.String("proposal.proposer", "local-validator"),
		},
	}
	x.emitSpans()
	spans := recorder.Ended()
	if len(spans) != 4 {
		t.Fatalf("got %d spans, want 4", len(spans))
	}
	for _, span := range spans[:2] {
		attrs := attribute.NewSet(span.Attributes()...)
		for key, want := range map[attribute.Key]string{
			"proposal.scope": "local", "proposal.value.hash_prefix": "loser", "proposal.proposer": "local-validator",
		} {
			got, ok := attrs.Value(key)
			if !ok || got.AsString() != want {
				t.Fatalf("%s: %s = %v, want %s", span.Name(), key, got, want)
			}
		}
	}
	attrs := attribute.NewSet(spans[3].Attributes()...)
	if got, _ := attrs.Value("value.hash_prefix"); got.AsString() != "winner" {
		t.Fatal("root lost committed value identity")
	}
}
