package sched

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thruputchain/chuckd/util/netutil"
	"github.com/thruputchain/chuckd/util/rpcutil"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/librpc"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// Test the actual RPC driver and generated protocol on loopback. Unimplemented
// RPCs panic if misrouted to this harness.
type testRPC struct {
	proto.SchedulerCoord1
	proposal func(context.Context) (proto.SchedulerProposalResult, error)
	null     func() error
	submit   func() error
}

func (s testRPC) Schedulercoord_null(context.Context) error { return s.null() }
func (s testRPC) Schedulercoord_submit_vetted(context.Context, proto.ValidatorID, proto.ParallelSignedTransactionBatch) error {
	return s.submit()
}
func (s testRPC) Schedulercoord_propose_block(ctx context.Context, _ proto.BlockID, _ proto.ProposalContext) (proto.SchedulerProposalResult, error) {
	return s.proposal(ctx)
}

func successfulProposal(context.Context) (proto.SchedulerProposalResult, error) {
	return proto.SchedulerProposalResult{Outcome: proto.SchedulerProposalError}, nil
}

// Run the ordinary SchedulerCoord RPC server on loopback.
type testListener struct {
	net.Listener
	cancel  context.CancelFunc
	mu      sync.Mutex
	sockets []*net.TCPConn
	wg      sync.WaitGroup
}

func startTestListener(t *testing.T, address string, handler func(*net.TCPConn) testRPC) *testListener {
	t.Helper()
	cfg := &net.ListenConfig{}
	ctx, cancel := context.WithCancel(context.Background())
	l, err := cfg.Listen(ctx, "tcp4", address)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	srv := &testListener{Listener: l, cancel: cancel}
	srv.wg.Add(1)
	go func() {
		defer srv.wg.Done()
		for {
			c, err := srv.Accept()
			if err != nil {
				return
			}
			sock := c.(*net.TCPConn)
			srv.mu.Lock()
			srv.sockets = append(srv.sockets, sock)
			srv.mu.Unlock()
			driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
			driver.Register(proto.SchedulerCoord1_Server{Srv: handler(sock)})
			srv.wg.Add(1)
			go func() { defer srv.wg.Done(); driver.Go() }()
		}
	}()
	t.Cleanup(func() { srv.Close() })
	return srv
}

func (s *testListener) Close() error {
	s.cancel()
	s.Listener.Close()
	s.wg.Wait()
	return nil
}

func (s *testListener) connectionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sockets)
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func awaitCount(t *testing.T, srv *testListener, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for srv.connectionCount() != count && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := srv.connectionCount(); got != count {
		t.Fatalf("accepted %d connections, want %d", got, count)
	}
}

func TestSharedPoolMixedRequests(t *testing.T) {
	var proposals, nulls, submits atomic.Int32
	srv := startTestListener(t, "127.0.0.1:0", func(*net.TCPConn) testRPC {
		return testRPC{
			null:   func() error { nulls.Add(1); return nil },
			submit: func() error { submits.Add(1); return nil },
			proposal: func(ctx context.Context) (proto.SchedulerProposalResult, error) {
				proposals.Add(1)
				return successfulProposal(ctx)
			},
		}
	})
	ctx := testContext(t)
	conn, err := ConnectPoolToScheduler(ctx, libnet.TCPNet{}, srv.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	awaitCount(t, srv, 20)
	if got := cap(conn.(*ConnPool).req); got != 20 {
		t.Fatalf("pool capacity = %d, want 20", got)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			if err := conn.Schedulercoord_submit_vetted(ctx, proto.ValidatorID{}, proto.ParallelSignedTransactionBatch{}); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer wg.Done()
			if err := conn.Schedulercoord_null(ctx); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer wg.Done()
			if r, err := conn.Schedulercoord_propose_block(ctx, 0, proto.ProposalContext{}); err != nil || r.Outcome != proto.SchedulerProposalError {
				t.Errorf("proposal: %v %v", r, err)
			}
		}()
	}
	wg.Wait()
	if proposals.Load() != 8 || nulls.Load() != 8 || submits.Load() != 8 {
		t.Fatalf("unexpected RPC counts: %d proposals %d nulls %d submits", proposals.Load(), nulls.Load(), submits.Load())
	}
}

// Two concurrent proposals must be able to use two shared workers. A
// dedicated proposal queue with a single worker cannot pass this test.
func TestConcurrentProposalsUseSharedWorkers(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	srv := startTestListener(t, "127.0.0.1:0", func(*net.TCPConn) testRPC {
		return testRPC{
			proposal: func(ctx context.Context) (proto.SchedulerProposalResult, error) {
				started <- struct{}{}
				<-release
				return successfulProposal(ctx)
			},
		}
	})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	ctx := testContext(t)
	conn, err := connectPoolToScheduler(ctx, libnet.TCPNet{}, srv.Addr().String(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			r, err := conn.Schedulercoord_propose_block(ctx, 0, proto.ProposalContext{})
			if err == nil && r.Outcome != proto.SchedulerProposalError {
				err = errors.New("unexpected proposal response")
			}
			results <- err
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-ctx.Done():
			t.Fatal("proposals did not start concurrently on shared workers")
		}
	}
	// Both RPCs reached separate server handlers before either could finish.
	// Release them and confirm both callers receive their replies.
	close(release)
	released = true
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}

func TestQueuedRequestCancelAndClose(t *testing.T) {
	ctx := testContext(t)
	started := make(chan struct{}, 1)
	unblock := make(chan struct{})
	var proposals atomic.Int32
	srv := startTestListener(t, "127.0.0.1:0", func(sock *net.TCPConn) testRPC {
		return testRPC{null: func() error { return nil }, proposal: func(ctx context.Context) (proto.SchedulerProposalResult, error) {
			proposals.Add(1)
			started <- struct{}{}
			<-unblock
			return successfulProposal(ctx)
		}}
	})
	defer close(unblock)
	conn, err := connectPoolToScheduler(ctx, libnet.TCPNet{}, srv.Addr().String(), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	result := make(chan error, 1)
	go func() {
		_, err := conn.Schedulercoord_propose_block(ctx, proto.BlockID(0), proto.ProposalContext{})
		result <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	canceled, cancel := context.WithTimeout(ctx, 25*time.Millisecond)
	defer cancel()
	if _, err := conn.Schedulercoord_propose_block(canceled, proto.BlockID(0), proto.ProposalContext{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued cancellation: %v", err)
	}
	conn.Close()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("in-flight proposal unexpectedly succeeded")
		}
	case <-ctx.Done():
		t.Fatal("Close left proposal blocked")
	}
	if _, err := conn.Schedulercoord_propose_block(ctx, proto.BlockID(0), proto.ProposalContext{}); err == nil {
		t.Fatal("closed pool accepted proposal")
	}
	if proposals.Load() != 1 {
		t.Fatalf("canceled proposal executed: %d", proposals.Load())
	}
}

func TestPoolStartupCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := connectPoolToScheduler(ctx, libnet.TCPNet{}, "127.0.0.1:1", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("startup cancellation: %v", err)
	}
	if _, err := connectPoolToScheduler(context.Background(), libnet.TCPNet{}, "127.0.0.1:1", 0); err == nil {
		t.Fatal("empty pool accepted")
	}
	empty := NewConnPool(context.Background(), nil)
	defer empty.Close()
	if err := empty.Schedulercoord_null(testContext(t)); err == nil {
		t.Fatal("empty pool accepted RPC")
	}
}

// All shared connections use the supplied dialer.
type trackedDialer struct {
	libnet.TCPNet
	calls atomic.Int32
}

func (d *trackedDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	d.calls.Add(1)
	return d.TCPNet.Dial(ctx, addr)
}

func TestPoolUsesCallerDialer(t *testing.T) {
	srv := startTestListener(t, "127.0.0.1:0", func(*net.TCPConn) testRPC {
		return testRPC{null: func() error { return nil }}
	})
	ctx := testContext(t)
	dialer := &trackedDialer{}
	conn, err := connectPoolToScheduler(ctx, dialer, srv.Addr().String(), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if dialer.calls.Load() != 2 {
		t.Fatalf("custom dialer received %d calls", dialer.calls.Load())
	}
	if err := conn.Schedulercoord_null(ctx); err != nil {
		t.Fatal(err)
	}
}
