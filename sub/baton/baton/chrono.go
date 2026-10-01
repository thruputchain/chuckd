package baton

import (
	"context"
	"sync"
	"time"

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
	emitted            bool
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
	if k.height != height {
		*k = blockChrono{height: height}
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

func (b *Baton) stampMempoolNominateEnd(height proto.BlockHeight, now time.Time) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	b.chrono.at(height).mempoolNominateEnd = now
	b.chrono.lastMempoolNominateEnd = now
	b.chrono.lastMempoolNominateEndHeight = height
}

func (b *Baton) reportSCPNominationDelay(ctx context.Context, height proto.BlockHeight, d time.Duration) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	b.chrono.at(height).scpNominationDelay = d
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

	b.chrono.at(height).persistCV = d
	b.chrono.lastPersistCV = d
	b.chrono.lastPersistCVHeight = height
}

func (b *Baton) stampNominateEnd(height proto.BlockHeight) {
	b.chrono.lk.Lock()
	defer b.chrono.lk.Unlock()

	now := b.handler.Now()
	b.chrono.at(height).nominateEnd = now
	b.chrono.at(height).emitSpans()
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
	b.chrono.at(height).commitEnd = now
	b.chrono.at(height).emitSpans()
	b.chrono.lastCommitEnd = now
	b.chrono.lastCommitEndHeight = height
}

// emitSpans waits for both concurrent paths to finish, then reconstructs the
// phase tree using the recorded clock values. Catchup has only a commit phase.
func (x *blockChrono) emitSpans() {
	if x.emitted || x.commitEnd.IsZero() || (!x.nominateStart.IsZero() && x.nominateEnd.IsZero()) {
		return
	}
	x.emitted = true
	start, end := x.commitStart, x.commitEnd
	if !x.nominateStart.IsZero() && x.nominateStart.Before(start) {
		start = x.nominateStart
	}
	if x.nominateEnd.After(end) {
		end = x.nominateEnd
	}
	tracer := otel.Tracer("github.com/groundhogchain/chuckd/sub/baton/baton")
	ctx, root := tracer.Start(context.Background(), "block", trace.WithNewRoot(),
		trace.WithTimestamp(start), trace.WithAttributes(append(x.attributes, attribute.Int64("block.height", int64(x.height)))...))
	phase := func(name string, start, end time.Time) {
		if start.IsZero() || end.IsZero() || end.Before(start) {
			return
		}
		_, span := tracer.Start(ctx, name, trace.WithTimestamp(start))
		span.End(trace.WithTimestamp(end))
	}
	phase("propose", x.nominateStart, x.mempoolNominateEnd)
	nominateStart := x.mempoolNominateEnd
	if nominateStart.IsZero() {
		nominateStart = x.nominateStart
	}
	phase("nominate", nominateStart, x.nominateEnd)
	phase("commit", x.commitStart, x.commitEnd)
	root.End(trace.WithTimestamp(end))
}
