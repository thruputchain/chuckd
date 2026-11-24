package sched

import (
	"context"
	"errors"
	"net"
	"runtime"
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

// Each accepted connection exposes its own negotiated transport to the RPC
// implementation. The test can close that actual socket after execution but
// before any success reply, simulating a lost response.
type testListener struct {
	net.Listener
	cancel  context.CancelFunc
	mu      sync.Mutex
	sockets []*net.TCPConn
	wg      sync.WaitGroup
}

func startTestListener(t *testing.T, address string, multipath bool, handler func(*net.TCPConn) testRPC) *testListener {
	t.Helper()
	cfg := &net.ListenConfig{}
	cfg.SetMultipathTCP(multipath)
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

func TestProposalConnectionRouting(t *testing.T) {
	for _, tc := range []struct {
		name           string
		client, server bool
		count          int
	}{
		{"disabled", false, false, 20},
		{"stock-server", true, false, 3},
		{"stock-client", false, true, 3},
		{"both-enabled", true, true, 20},
		{"single-connection", true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.client && runtime.GOOS != "linux" {
				t.Skip("Linux proposal transport")
			}
			var proposals, nulls, submits, mptcpProposals, mptcpOther atomic.Int32
			srv := startTestListener(t, "127.0.0.1:0", tc.server, func(sock *net.TCPConn) testRPC {
				return testRPC{
					null: func() error {
						nulls.Add(1)
						if active, _ := sock.MultipathTCP(); active {
							mptcpOther.Add(1)
						}
						return nil
					},
					submit: func() error {
						submits.Add(1)
						if active, _ := sock.MultipathTCP(); active {
							mptcpOther.Add(1)
						}
						return nil
					},
					proposal: func(ctx context.Context) (proto.SchedulerProposalResult, error) {
						proposals.Add(1)
						if active, _ := sock.MultipathTCP(); active {
							mptcpProposals.Add(1)
						}
						return successfulProposal(ctx)
					},
				}
			})
			ctx := testContext(t)
			// Independently establish whether the loopback route supports MPTCP so a
			// broken implementation cannot masquerade as legitimate TCP fallback.
			probeCount := 0
			wantMPTCP := false
			if tc.client && tc.server && tc.count > 1 {
				dialer := net.Dialer{}
				dialer.SetMultipathTCP(true)
				probe, err := dialer.DialContext(ctx, "tcp4", srv.Addr().String())
				if err != nil {
					t.Fatal(err)
				}
				wantMPTCP, err = probe.(*net.TCPConn).MultipathTCP()
				if err != nil {
					t.Fatal(err)
				}
				probe.Close()
				probeCount++
			}
			var conn SchedulerCoordConn
			var err error
			if tc.count == 20 {
				enabled := ""
				if tc.client {
					enabled = "1"
				}
				t.Setenv("HOGD_COORD_PROPOSAL_MPTCP", enabled)
				conn, err = ConnectPoolToScheduler(ctx, libnet.TCPNet{}, srv.Addr().String())
			} else {
				conn, err = connectPoolToScheduler(ctx, libnet.TCPNet{}, srv.Addr().String(), tc.count, tc.client)
			}
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(conn.Close)
			awaitCount(t, srv, tc.count+probeCount)
			p := conn.(*ConnPool)
			dedicated := p.req != p.proposalReq
			if dedicated != (tc.client && tc.count > 1) {
				t.Fatalf("dedicated queue=%t", dedicated)
			}
			ordinaryCount := tc.count
			if dedicated {
				ordinaryCount--
			}
			if got := cap(p.req); got != ordinaryCount {
				t.Fatalf("ordinary capacity=%d", got)
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
					if r, err := conn.Schedulercoord_propose_block(ctx, proto.BlockID(0), proto.ProposalContext{}); err != nil || r.Outcome != proto.SchedulerProposalError {
						t.Errorf("proposal: %v %v", r, err)
					}
				}()
			}
			wg.Wait()
			if proposals.Load() != 8 || nulls.Load() != 8 || submits.Load() != 8 {
				t.Fatalf("unexpected RPC counts: %d proposals %d nulls %d submits", proposals.Load(), nulls.Load(), submits.Load())
			}
			if mptcpOther.Load() != 0 {
				t.Fatal("non-proposal RPC negotiated MPTCP")
			}
			if wantMPTCP {
				if mptcpProposals.Load() != 8 {
					t.Fatal("proposal RPCs did not negotiate MPTCP")
				}
			} else if mptcpProposals.Load() != 0 {
				t.Fatal("unexpected MPTCP proposal")
			} else if tc.client && tc.server && tc.count > 1 {
				t.Log("MPTCP unavailable; verified TCP fallback")
			}

			// The ordinary single-connection client API is unchanged and compatible
			// with an MPTCP-capable server.
			direct, err := DialSchedulerCoordClient(ctx, libnet.TCPNet{}, srv.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer direct.Close()
			if err := direct.Schedulercoord_null(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProposalReconnectDoesNotReplay(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux proposal transport")
	}
	var proposals atomic.Int32
	handler := func(sock *net.TCPConn) testRPC {
		return testRPC{null: func() error { return nil }, proposal: func(ctx context.Context) (proto.SchedulerProposalResult, error) {
			if proposals.Add(1) == 1 {
				sock.Close()
			}
			return successfulProposal(ctx)
		}}
	}
	srv := startTestListener(t, "127.0.0.1:0", true, handler)
	addr := srv.Addr().String()
	ctx := testContext(t)
	conn, err := connectPoolToScheduler(ctx, libnet.TCPNet{}, addr, 2, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Schedulercoord_propose_block(ctx, proto.BlockID(0), proto.ProposalContext{}); err == nil {
		t.Fatal("lost response reported success")
	}
	if proposals.Load() != 1 {
		t.Fatalf("first proposal replayed %d times", proposals.Load())
	}
	if err := conn.Schedulercoord_null(ctx); err != nil {
		t.Fatalf("ordinary pool failed: %v", err)
	}
	srv.Close()
	if _, err := conn.Schedulercoord_propose_block(ctx, proto.BlockID(0), proto.ProposalContext{}); err == nil {
		t.Fatal("proposal succeeded during outage")
	}
	if proposals.Load() != 1 {
		t.Fatal("proposal executed during outage")
	}
	replacement := startTestListener(t, addr, false, handler)
	if r, err := conn.Schedulercoord_propose_block(ctx, proto.BlockID(0), proto.ProposalContext{}); err != nil || r.Outcome != proto.SchedulerProposalError {
		t.Fatalf("new proposal after restart/fallback: %v %v", r, err)
	}
	awaitCount(t, replacement, 1)
	if proposals.Load() != 2 {
		t.Fatalf("want two executed proposals, got %d", proposals.Load())
	}
}

func TestProposalCancelAndClose(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux proposal transport")
	}
	ctx := testContext(t)
	started := make(chan struct{}, 1)
	unblock := make(chan struct{})
	var proposals atomic.Int32
	srv := startTestListener(t, "127.0.0.1:0", false, func(sock *net.TCPConn) testRPC {
		return testRPC{null: func() error { return nil }, proposal: func(ctx context.Context) (proto.SchedulerProposalResult, error) {
			proposals.Add(1)
			started <- struct{}{}
			<-unblock
			return successfulProposal(ctx)
		}}
	})
	defer close(unblock)
	conn, err := connectPoolToScheduler(ctx, libnet.TCPNet{}, srv.Addr().String(), 2, true)
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
	if err := conn.Schedulercoord_null(ctx); err != nil {
		t.Fatalf("null while proposal blocked: %v", err)
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
	if _, err := connectPoolToScheduler(ctx, libnet.TCPNet{}, "127.0.0.1:1", 2, true); !errors.Is(err, context.Canceled) {
		t.Fatalf("startup cancellation: %v", err)
	}
	if _, err := connectPoolToScheduler(context.Background(), libnet.TCPNet{}, "127.0.0.1:1", 0, true); err == nil {
		t.Fatal("empty pool accepted")
	}
	empty := NewConnPool(context.Background(), nil)
	defer empty.Close()
	if err := empty.Schedulercoord_null(testContext(t)); err == nil {
		t.Fatal("empty pool accepted RPC")
	}
}

// A custom network must not be bypassed even when the application opts in.
type trackedDialer struct {
	libnet.TCPNet
	calls atomic.Int32
}

func (d *trackedDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	d.calls.Add(1)
	return d.TCPNet.Dial(ctx, addr)
}

func TestOptInKeepsCustomDialer(t *testing.T) {
	srv := startTestListener(t, "127.0.0.1:0", false, func(*net.TCPConn) testRPC {
		return testRPC{null: func() error { return nil }}
	})
	ctx := testContext(t)
	dialer := &trackedDialer{}
	conn, err := connectPoolToScheduler(ctx, dialer, srv.Addr().String(), 2, true)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if dialer.calls.Load() != 2 {
		t.Fatalf("custom dialer received %d calls", dialer.calls.Load())
	}
	if p := conn.(*ConnPool); p.req != p.proposalReq {
		t.Fatal("custom dialer unexpectedly bypassed for proposal")
	}
	if err := conn.Schedulercoord_null(ctx); err != nil {
		t.Fatal(err)
	}
}
