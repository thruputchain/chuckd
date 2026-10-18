package baton

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/thruputchain/chuckd/sub/baton/batonkit"
	"github.com/thruputchain/chuckd/util/blocktrace"
	proto "github.com/thruputchain/thruput-protocol-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newRecorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	r := tracetest.NewSpanRecorder()
	p := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(r),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), sdktrace.WithIDGenerator(blocktrace.IDGenerator()))
	old := otel.GetTracerProvider()
	otel.SetTracerProvider(p)
	t.Cleanup(func() {
		otel.SetTracerProvider(old)
		if err := p.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return r
}

type chronoHandler struct {
	batonkit.Handler
	now time.Time
}

func (h *chronoHandler) Now() time.Time { return h.now }

func newChronoBaton() (*Baton, *chronoHandler) {
	h := &chronoHandler{now: time.Unix(100, 0)}
	return &Baton{handler: h}, h
}

func proposalAttrs() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("proposal.value.hash_prefix", "loser"),
		attribute.String("proposal.proposer", "local-validator"),
	}
}

func setCommitIdentity(b *Baton, height proto.BlockHeight, hash [32]byte) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()
	if x := b.chrono.existing(height); x != nil {
		x.attributes = []attribute.KeyValue{attribute.String("value.hash_prefix", "winner")}
		x.networkHash, x.hasNetworkHash = hash, true
	}
}

func spanTree(t *testing.T, r *tracetest.SpanRecorder, count int) map[string]sdktrace.ReadOnlySpan {
	t.Helper()
	spans := r.Ended()
	if len(spans) != count {
		t.Fatalf("got %d spans, want %d", len(spans), count)
	}
	byName := make(map[string]sdktrace.ReadOnlySpan)
	for _, span := range spans {
		if byName[span.Name()] != nil {
			t.Fatalf("duplicate phase %s", span.Name())
		}
		byName[span.Name()] = span
	}
	root := byName["block"]
	if root == nil || root.Parent().IsValid() {
		t.Fatal("missing or incorrectly parented root")
	}
	for name, span := range byName {
		if name != "block" && (span.Parent().SpanID() != root.SpanContext().SpanID() ||
			span.SpanContext().TraceID() != root.SpanContext().TraceID()) {
			t.Fatalf("incorrect parent or trace ID for %s", name)
		}
		attrs := attribute.NewSet(span.Attributes()...)
		if got, ok := attrs.Value("groundhog.block.height"); !ok || got.AsInt64() != 42 {
			t.Fatalf("%s missing groundhog.block.height", name)
		}
		if attrs.HasValue("proposal.scope") {
			t.Fatal("obsolete proposal.scope attribute")
		}
		if name == "propose" || name == "nominate" {
			for _, want := range proposalAttrs() {
				got, ok := attrs.Value(want.Key)
				if !ok || got != want.Value {
					t.Fatalf("%s missing proposal attribute %s", name, want.Key)
				}
			}
		}
	}
	return byName
}

func TestBlockSpanTree(t *testing.T) {
	r := newRecorder(t)
	b, clock := newChronoBaton()
	t0 := clock.now
	hash := [32]byte{0x11}
	b.stampNominateStart(context.Background(), 42)
	clock.now = t0.Add(time.Second)
	b.stampMempoolNominateEnd(42, clock.now, proposalAttrs(), hash)
	clock.now = t0.Add(2 * time.Second)
	b.stampNominateEnd(42)
	if len(r.Ended()) != 0 {
		t.Fatal("emitted before commit")
	}
	clock.now = t0.Add(3 * time.Second)
	b.stampCommitStart(context.Background(), 42)
	setCommitIdentity(b, 42, hash)
	clock.now = t0.Add(4 * time.Second)
	b.stampCommitEnd(42)
	b.stampCommitEnd(42)
	spans := spanTree(t, r, 4)
	root := spans["block"]
	if root.SpanContext().TraceID() != blocktrace.TraceID(hash, 42) ||
		!root.StartTime().Equal(t0) || !root.EndTime().Equal(clock.now) {
		t.Fatal("incorrect root identity or timestamps")
	}
	attrs := attribute.NewSet(root.Attributes()...)
	if got, ok := attrs.Value("groundhog.block.height"); !ok || got.AsInt64() != 42 {
		t.Fatal("incorrect block height")
	}
	if got, _ := attrs.Value("value.hash_prefix"); got.AsString() != "winner" {
		t.Fatal("root lost committed value identity")
	}
}

func TestCommitSpanSurvivesPendingNomination(t *testing.T) {
	r := newRecorder(t)
	b, clock := newChronoBaton()
	t0 := clock.now
	hash := [32]byte{2}
	b.stampNominateStart(context.Background(), 42)
	clock.now = t0.Add(time.Second)
	b.stampMempoolNominateEnd(42, clock.now, proposalAttrs(), hash)
	clock.now = t0.Add(2 * time.Second)
	b.stampCommitStart(context.Background(), 42)
	setCommitIdentity(b, 42, hash)
	clock.now = t0.Add(3 * time.Second)
	b.stampCommitEnd(42)
	spans := spanTree(t, r, 3)
	if spans["nominate"] != nil || !spans["block"].EndTime().Equal(clock.now) {
		t.Fatal("incorrect pending nomination emission")
	}
	clock.now = t0.Add(4 * time.Second)
	b.stampNominateEnd(42)
	b.stampNominateEnd(42)
	b.stampMempoolNominateEnd(42, t0.Add(time.Second), proposalAttrs(), hash)
	spans = spanTree(t, r, 4)
	if spans["nominate"] == nil || spans["block"].SpanContext().TraceID() != blocktrace.TraceID(hash, 42) {
		t.Fatal("late nomination lost trace identity")
	}
}

func TestStaleChronoWrites(t *testing.T) {
	r := newRecorder(t)
	b, clock := newChronoBaton()
	if b.chrono.existing(0) != nil {
		t.Fatal("unclaimed zero-height slot is owned")
	}
	b.stampNominateStart(context.Background(), 42)
	clock.now = clock.now.Add(time.Second)
	b.stampNominateStart(context.Background(), 44)
	b.stampMempoolNominateEnd(42, clock.now, proposalAttrs(), [32]byte{1})
	b.stampNominateEnd(42)
	b.reportSCPNominationDelay(context.Background(), 42, time.Second)
	b.reportPersistCV(42, time.Second)
	b.stampCommitEnd(42)
	x := b.chrono.existing(44)
	if x == nil || !x.nominateStart.Equal(clock.now) || !x.mempoolNominateEnd.IsZero() ||
		!x.nominateEnd.IsZero() || !x.commitEnd.IsZero() || x.scpNominationDelay != 0 || x.persistCV != 0 || x.hasNetworkHash {
		t.Fatal("stale completion overwrote reused slot")
	}
	if len(r.Ended()) != 0 || b.chrono.lastNominateEndHeight != 42 || b.chrono.lastCommitEndHeight != 42 {
		t.Fatal("incorrect stale completion bookkeeping")
	}
}

func TestProposalAttributesInterleave(t *testing.T) {
	for i := range 200 {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := newRecorder(t)
			b, clock := newChronoBaton()
			hash := [32]byte{3}
			b.stampNominateStart(context.Background(), 42)
			mempoolEnd := clock.now.Add(time.Second)
			clock.now = clock.now.Add(2 * time.Second)
			b.stampCommitStart(context.Background(), 42)
			setCommitIdentity(b, 42, hash)
			clock.now = clock.now.Add(time.Second)
			var wg sync.WaitGroup
			start := make(chan struct{})
			wg.Add(2)
			go func() { defer wg.Done(); <-start; b.stampMempoolNominateEnd(42, mempoolEnd, proposalAttrs(), hash) }()
			go func() { defer wg.Done(); <-start; b.stampCommitEnd(42) }()
			close(start)
			wg.Wait()
			clock.now = clock.now.Add(time.Second)
			b.stampNominateEnd(42)
			spanTree(t, r, 4)
		})
	}
}

func TestCatchupOnlySpans(t *testing.T) {
	r := newRecorder(t)
	b, clock := newChronoBaton()
	t0 := clock.now
	b.stampCommitStart(context.Background(), 42)
	clock.now = t0.Add(time.Second)
	b.stampCommitEnd(42)
	spans := spanTree(t, r, 2)
	if spans["commit"] == nil || !spans["block"].SpanContext().IsValid() ||
		!spans["block"].StartTime().Equal(t0) || !spans["block"].EndTime().Equal(clock.now) {
		t.Fatal("incorrect catchup trace")
	}
}
