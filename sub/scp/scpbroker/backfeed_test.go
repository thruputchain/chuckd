package scpbroker

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thruputchain/chuckd/sub/scp/scpkit"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type hangingRepairer struct {
	calls atomic.Int32
}

func (r *hangingRepairer) RepairMissingSCP(ctx context.Context, slot scpkit.Slot) (proto.SCPEnvelopes, error) {
	r.calls.Add(1)
	<-ctx.Done()
	return nil, ctx.Err()
}

type dialHangingRepairer struct {
	started chan struct{}
}

func (r *dialHangingRepairer) RepairMissingSCP(ctx context.Context, slot scpkit.Slot) (proto.SCPEnvelopes, error) {
	close(r.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

type replyHangingRepairer struct {
	started   chan struct{}
	connected chan struct{}
}

func (r *replyHangingRepairer) RepairMissingSCP(ctx context.Context, slot scpkit.Slot) (proto.SCPEnvelopes, error) {
	close(r.started)
	close(r.connected) // Simulate a successful connection before hanging on RPC reply.
	<-ctx.Done()
	return nil, ctx.Err()
}

func waitForBackfeedTimeout(t *testing.T, done <-chan struct{}, timeout time.Duration) {
	t.Helper()

	select {
	case <-done:
		t.Fatal("fulfillBackfeed returned before timeout expired")
	case <-time.After(timeout / 2):
	}

	select {
	case <-done:
	case <-time.After(timeout + 500*time.Millisecond):
		t.Fatal("fulfillBackfeed did not return after timeout expired")
	}
}

func TestBackfeedTimeout(t *testing.T) {
	t.Parallel()

	repairer := &hangingRepairer{}
	broker := NewSCPBroker(context.Background(), nil, nil, repairer)

	before := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		broker.SCPBackfeedEnvelopes(context.Background(), scpkit.Slot(i), 100*time.Millisecond)
	}

	// Wait for goroutines to start
	deadline := time.Now().Add(time.Second)
	for repairer.calls.Load() < 5 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if repairer.calls.Load() < 5 {
		t.Fatal("not all backfeed goroutines started")
	}

	// Wait for timeout to expire (100ms timeout + generous margin)
	time.Sleep(500 * time.Millisecond)

	after := runtime.NumGoroutine()
	leaked := after - before
	if leaked > 2 {
		t.Errorf("goroutine leak: %d goroutines still running after timeout", leaked)
	}
}

func TestBackfeedStopCancels(t *testing.T) {
	t.Parallel()

	repairer := &hangingRepairer{}
	broker := NewSCPBroker(context.Background(), nil, nil, repairer)

	before := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		broker.SCPBackfeedEnvelopes(context.Background(), scpkit.Slot(i), time.Minute)
	}

	// Wait for goroutines to start
	deadline := time.Now().Add(time.Second)
	for repairer.calls.Load() < 5 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if repairer.calls.Load() < 5 {
		t.Fatal("not all backfeed goroutines started")
	}

	broker.Stop()

	// Goroutines should terminate promptly after Stop
	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	leaked := after - before
	if leaked > 2 {
		t.Errorf("goroutine leak after Stop: %d goroutines still running", leaked)
	}
}

func TestFulfillBackfeedTimesOutWhenConnectHangs(t *testing.T) {
	t.Parallel()

	timeout := 100 * time.Millisecond
	repairer := &dialHangingRepairer{started: make(chan struct{})}
	broker := NewSCPBroker(context.Background(), nil, nil, repairer)

	done := make(chan struct{})
	go func() {
		broker.fulfillBackfeed(context.Background(), 1, timeout)
		close(done)
	}()

	select {
	case <-repairer.started:
	case <-time.After(time.Second):
		t.Fatal("repair call did not start")
	}

	waitForBackfeedTimeout(t, done, timeout)
}

func TestFulfillBackfeedTimesOutWhenRPCReplyHangs(t *testing.T) {
	t.Parallel()

	timeout := 100 * time.Millisecond
	repairer := &replyHangingRepairer{
		started:   make(chan struct{}),
		connected: make(chan struct{}),
	}
	broker := NewSCPBroker(context.Background(), nil, nil, repairer)

	done := make(chan struct{})
	go func() {
		broker.fulfillBackfeed(context.Background(), 1, timeout)
		close(done)
	}()

	select {
	case <-repairer.started:
	case <-time.After(time.Second):
		t.Fatal("repair call did not start")
	}

	select {
	case <-repairer.connected:
	case <-time.After(time.Second):
		t.Fatal("repair call did not reach connected state")
	}

	waitForBackfeedTimeout(t, done, timeout)
}
