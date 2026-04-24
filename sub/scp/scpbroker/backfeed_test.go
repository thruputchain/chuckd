package scpbroker

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	proto "github.com/groundhogchain/protocol-go"
)

type hangingRepairer struct {
	calls atomic.Int32
}

func (r *hangingRepairer) RepairMissingSCP(ctx context.Context, slot scpkit.Slot) (proto.SCPEnvelopes, error) {
	r.calls.Add(1)
	<-ctx.Done()
	return nil, ctx.Err()
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
