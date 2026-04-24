package scpbroker_test

import (
	"context"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/sub/scp/scpbroker"
	"github.com/groundhogchain/chuckd/sub/scp/scpd"
	"github.com/groundhogchain/chuckd/test/testnet"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
)

// mockSCPControl implements proto.SCP_Control1 with a gate on received_envelope.
// When received_envelope is called, it signals arrived and blocks until gate is closed.
type mockSCPControl struct {
	gate    chan struct{}
	arrived chan struct{}
}

func (m *mockSCPControl) Scp_control_null(context.Context) error { return nil }
func (m *mockSCPControl) Scp_control_nominate(context.Context, proto.SCPSlotIndex, proto.SCPValue, proto.SCPValue) (bool, error) {
	return false, nil
}
func (m *mockSCPControl) Scp_control_stop_nomination(context.Context, proto.SCPSlotIndex) error {
	return nil
}
func (m *mockSCPControl) Scp_control_update_local_quorum_set(context.Context, proto.SCPQuorumSet) error {
	return nil
}
func (m *mockSCPControl) Scp_control_received_envelope(_ context.Context, _ proto.SCPEnvelope) error {
	select {
	case <-m.arrived:
	default:
		close(m.arrived)
	}
	<-m.gate
	return nil
}
func (m *mockSCPControl) Scp_control_set_state_from_envelope(context.Context, proto.SCPSlotIndex, proto.SCPEnvelope) error {
	panic("unused")
}
func (m *mockSCPControl) Scp_control_set_qset(context.Context, proto.SCPQuorumSet) (proto.SCPOptQuorumSet, error) {
	return nil, nil
}
func (m *mockSCPControl) Scp_control_get_qset(context.Context, proto.SCPHash) (proto.SCPOptQuorumSet, error) {
	return nil, nil
}
func (m *mockSCPControl) Scp_control_remove_qset(context.Context, proto.SCPHash) (proto.SCPOptQuorumSet, error) {
	return nil, nil
}
func (m *mockSCPControl) Scp_control_set_lowest_non_externalized_slot(context.Context, proto.SCPSlotIndex) error {
	return nil
}

type noopStorage struct{}

func (noopStorage) PersistSCPEnvelope(context.Context, proto.SCPEnvelope) {}
func (noopStorage) FindSCPEnvelopesForSlot(context.Context, proto.SCPSlotIndex) []proto.SCPEnvelope {
	return nil
}

// TestBrokerStopDrainsInflightCalls verifies that Stop() blocks until all
// in-flight controller calls complete, so that controller.Close() after Stop()
// never hits an active RPC.
func TestBrokerStopDrainsInflightCalls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gate := make(chan struct{})
	arrived := make(chan struct{})
	mock := &mockSCPControl{gate: gate, arrived: arrived}

	tn := testnet.NewTestNetwork()
	addr := "test-scp-control"
	l := librpc.StartSerial(ctx, tn, netutil.MaxMessageSize, rpcutil.BaseDriverRegime, addr,
		proto.SCP_Control1_Server{Srv: mock})
	defer l.Close()

	sock, err := tn.DialAddress(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	controller := scpd.NewSCPController(ctx, sock)

	broker := scpbroker.NewSCPBroker(ctx, nil, noopStorage{}, nil)
	broker.SetSCPController(controller)

	env := proto.SCPEnvelope{Statement: proto.SCPStatement{Slot_index: 1}}

	// start an in-flight ReceivedEnvelope call — it will block on the mock server
	go broker.SCPFeedEnvelope(ctx, env)
	<-arrived

	// Stop() should block until the in-flight call completes
	stopped := make(chan struct{})
	go func() {
		broker.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("Stop() returned while a controller call was still in-flight")
	case <-time.After(50 * time.Millisecond):
	}

	// unblock the server — the in-flight call finishes, then Stop() returns
	close(gate)

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop() did not return after in-flight call completed")
	}

	// this is the line that panicked before the fix — safe now because Stop() drained
	controller.Close()

	// cancel context so driver goroutines shut down before the test exits
	cancel()
}
