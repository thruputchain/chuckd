package baton

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/util/blocktrace"
	"github.com/groundhogchain/lib-go/liblog"
	proto "github.com/groundhogchain/protocol-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type blockChrono struct {
	height             proto.BlockHeight
	nominateStart      time.Time
	mempoolNominateEnd time.Time
	nominateEnd        time.Time
	commitStart        time.Time
	commitEnd          time.Time
	persistCV          time.Duration
	scpNominationDelay time.Duration
	attributes         []attribute.KeyValue
	proposalAttributes []attribute.KeyValue
	owned              bool
	networkHash        [32]byte
	hasNetworkHash     bool
	rootSC             trace.SpanContext
	rootEmitted        bool
	proposeEmitted     bool
	nominateEmitted    bool
	commitEmitted      bool
}

type chrono struct {
	lk    sync.Mutex
	block [2]blockChrono
	//
	lastNominateStart      time.Time
	lastMempoolNominateEnd time.Time
	lastNominateEnd        time.Time
	lastCommitStart        time.Time
	lastCommitEnd          time.Time
	lastPersistCV          time.Duration
	lastSCPNominationDelay time.Duration
	//
	lastNominateStartHeight      proto.BlockHeight
	lastMempoolNominateEndHeight proto.BlockHeight
	lastNominateEndHeight        proto.BlockHeight
	lastCommitStartHeight        proto.BlockHeight
	lastCommitEndHeight          proto.BlockHeight
	lastPersistCVHeight          proto.BlockHeight
	lastSCPNominationDelayHeight proto.BlockHeight
}

func (x *chrono) at(height proto.BlockHeight) *blockChrono {
	k := &x.block[int(height)%len(x.block)]
	if !k.owned || k.height != height {
		*k = blockChrono{height: height, owned: true}
	}
	return k
}

func (x *chrono) existing(height proto.BlockHeight) *blockChrono {
	k := &x.block[int(height)%len(x.block)]
	if !k.owned || k.height != height {
		return nil
	}
	return k
}

func (b *Baton) stampNominateStart(ctx context.Context, height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).nominateStart = now
	b.chrono.lastNominateStart = now
	b.chrono.lastNominateStartHeight = height

	if !b.chrono.lastCommitStart.IsZero() {
		liblog.Debugf(ctx, "commit_start_to_nominate_start_delay=%v | commit_height=%d, nominate_height=%d",
			b.chrono.lastNominateStart.Sub(b.chrono.lastCommitStart),
			b.chrono.lastCommitStartHeight, b.chrono.lastNominateStartHeight,
		)
	}
}

func (b *Baton) stampMempoolNominateEnd(height proto.BlockHeight, now time.Time, attrs []attribute.KeyValue, networkHash [32]byte) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	if x := b.chrono.existing(height); x != nil {
		x.mempoolNominateEnd = now
		x.proposalAttributes = attrs
		x.networkHash, x.hasNetworkHash = networkHash, true
		x.emitSpans()
	}
	b.chrono.lastMempoolNominateEnd = now
	b.chrono.lastMempoolNominateEndHeight = height
}

func (b *Baton) reportSCPNominationDelay(ctx context.Context, height proto.BlockHeight, d time.Duration) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	if x := b.chrono.existing(height); x != nil {
		x.scpNominationDelay = d
	}
	b.chrono.lastSCPNominationDelay = d
	b.chrono.lastSCPNominationDelayHeight = height

	// single-node
	// |----- scp nomination -----|
	// |----- persist ------|
	//
	// multi-node
	// |-- scp nomination --|
	// |------------ persist ------------|
	if b.chrono.lastPersistCVHeight == b.chrono.lastSCPNominationDelayHeight {
		liblog.Debugf(ctx, "coordinator_non_persist_overhead=%v | height=%d",
			b.chrono.lastSCPNominationDelay-b.chrono.lastPersistCV,
			b.chrono.lastPersistCVHeight)
	}
}

func (b *Baton) reportPersistCV(height proto.BlockHeight, d time.Duration) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	if x := b.chrono.existing(height); x != nil {
		x.persistCV = d
	}
	b.chrono.lastPersistCV = d
	b.chrono.lastPersistCVHeight = height
}

func (b *Baton) stampNominateEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	if x := b.chrono.existing(height); x != nil {
		x.nominateEnd = now
		x.emitSpans()
	}
	b.chrono.lastNominateEnd = now
	b.chrono.lastNominateEndHeight = height
}

func (b *Baton) stampCommitStart(ctx context.Context, height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).commitStart = now
	b.chrono.lastCommitStart = now
	b.chrono.lastCommitStartHeight = height

	if !b.chrono.lastNominateStart.IsZero() {
		liblog.Debugf(ctx, "nominate_start_to_commit_start_delay=%v | nominate_height=%d, commit_height=%d",
			b.chrono.lastCommitStart.Sub(b.chrono.lastNominateStart),
			b.chrono.lastNominateStartHeight, b.chrono.lastCommitStartHeight,
		)
	}

	if !b.chrono.lastMempoolNominateEnd.IsZero() {
		liblog.Debugf(ctx, "mempool_nominate_end_to_commit_start_delay=%v | nominate_height=%d, commit_height=%d",
			b.chrono.lastCommitStart.Sub(b.chrono.lastMempoolNominateEnd),
			b.chrono.lastMempoolNominateEndHeight, b.chrono.lastCommitStartHeight,
		)
	}
}

func (b *Baton) stampCommitEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	if x := b.chrono.existing(height); x != nil {
		x.commitEnd = now
		x.emitSpans()
	}
	b.chrono.lastCommitEnd = now
	b.chrono.lastCommitEndHeight = height
}

// emitSpans reconstructs completed phases as soon as commit finishes. A local
// proposal may still be running; it must not delay emission of the commit trace.
// Phases that complete later are added as children of the already-ended root,
// so they can extend past the root's end. Catchup has only a commit phase.
func (x *blockChrono) emitSpans() {
	if x.commitEnd.IsZero() {
		return
	}
	start, end := x.commitStart, x.commitEnd
	if !x.nominateStart.IsZero() && x.nominateStart.Before(start) {
		start = x.nominateStart
	}
	if x.nominateEnd.After(end) {
		end = x.nominateEnd
	}
	tracer := otel.Tracer("github.com/groundhogchain/chuckd/sub/baton/baton")
	// Late phases remain children of the original root even after it has ended.
	ctx := trace.ContextWithSpanContext(context.Background(), x.rootSC)
	if !x.rootEmitted {
		rootCtx := context.Background()
		if x.hasNetworkHash {
			rootCtx = blocktrace.WithTraceID(rootCtx, blocktrace.TraceID(x.networkHash, uint64(x.height)))
		}
		attrs := append([]attribute.KeyValue{}, x.attributes...)
		attrs = append(attrs, attribute.Int64("groundhog.block.height", int64(x.height)))
		var root trace.Span
		ctx, root = tracer.Start(rootCtx, "block", trace.WithNewRoot(),
			trace.WithTimestamp(start), trace.WithAttributes(attrs...))
		x.rootSC, x.rootEmitted = root.SpanContext(), true
		defer root.End(trace.WithTimestamp(end))
	}
	phase := func(name string, start, end time.Time, emitted *bool, attrs ...attribute.KeyValue) {
		if *emitted || start.IsZero() || end.IsZero() || end.Before(start) {
			return
		}
		_, span := tracer.Start(ctx, name, trace.WithTimestamp(start), trace.WithAttributes(attrs...))
		span.End(trace.WithTimestamp(end))
		*emitted = true
	}
	phase("propose", x.nominateStart, x.mempoolNominateEnd, &x.proposeEmitted, x.proposalAttributes...)
	nominateStart := x.mempoolNominateEnd
	if nominateStart.IsZero() {
		nominateStart = x.nominateStart
	}
	phase("nominate", nominateStart, x.nominateEnd, &x.nominateEmitted, x.proposalAttributes...)
	phase("commit", x.commitStart, x.commitEnd, &x.commitEmitted)
}
