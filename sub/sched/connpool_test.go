package sched

import (
	"context"
	"errors"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

// outcomeConn answers Schedcoord_lookup_transaction_outcomes from a fixed
// table; every other SchedulerCoord1 method is left unimplemented.
type outcomeConn struct {
	proto.SchedulerCoord1
	pending map[proto.TransactionHash]bool
	err     error
}

func (c *outcomeConn) Close() {}

func (c *outcomeConn) Schedcoord_lookup_transaction_outcomes(
	ctx context.Context,
	hashes proto.TransactionHashBatch,
) (proto.SchedulerTransactionOutcomeBatch, error) {
	if c.err != nil {
		return nil, c.err
	}
	outcomes := make(proto.SchedulerTransactionOutcomeBatch, len(hashes))
	for i, h := range hashes {
		if c.pending[h] {
			outcomes[i].Status = proto.SchedulerTransactionPending
		} else {
			outcomes[i].Status = proto.SchedulerTransactionNotFound
		}
	}
	return outcomes, nil
}

func TestConnPoolLookupTransactionOutcomes(t *testing.T) {
	ctx := context.Background()
	known := proto.TransactionHash{1}
	unknown := proto.TransactionHash{2}

	pool := NewConnPool(ctx, []SchedulerCoordConn{
		&outcomeConn{pending: map[proto.TransactionHash]bool{known: true}},
	})
	defer pool.Close()

	got, err := pool.Schedcoord_lookup_transaction_outcomes(
		ctx, proto.TransactionHashBatch{known, unknown, known})
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}
	want := []proto.SchedulerTransactionStatus{
		proto.SchedulerTransactionPending,
		proto.SchedulerTransactionNotFound,
		proto.SchedulerTransactionPending,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d outcomes, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Status != want[i] {
			t.Errorf("outcome %d: got status %v, want %v", i, got[i].Status, want[i])
		}
	}
}

func TestConnPoolLookupTransactionOutcomesReturnsSchedulerError(t *testing.T) {
	ctx := context.Background()
	schedErr := errors.New("scheduler unavailable")

	pool := NewConnPool(ctx, []SchedulerCoordConn{&outcomeConn{err: schedErr}})
	defer pool.Close()

	_, err := pool.Schedcoord_lookup_transaction_outcomes(
		ctx, proto.TransactionHashBatch{{1}})
	if !errors.Is(err, schedErr) {
		t.Fatalf("got error %v, want %v", err, schedErr)
	}
}
