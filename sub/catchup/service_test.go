package catchup

import (
	"context"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/sub/storage"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/librand"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type NoopCatchupPersistence struct{}

var _ storage.CatchupBlockPersistence = &NoopCatchupPersistence{}

func NewNoopCatchupPersistence() *NoopCatchupPersistence {
	return &NoopCatchupPersistence{}
}

func (m *NoopCatchupPersistence) PrePersistCatchupBlock(context.Context, proto.CatchupBlock, proto.ConsensusValueHash) {
}

func (m *NoopCatchupPersistence) PersistAlreadyPrePersistedCatchupBlock(context.Context, proto.BlockHeight, proto.ConsensusValueHash) {
}

func (m *NoopCatchupPersistence) FindCatchupBlock(context.Context, proto.BlockHeight) *proto.CatchupBlock {
	return nil
}

func signedRequest(t *testing.T, spans []proto.CatchupSpan) proto.SignedCatchupBlocksRequest {
	t.Helper()
	ctx := context.Background()
	rand := librand.NewRand()
	attacker := protoutil.MustGenerateIdentity(ctx, rand, "attacker:9999")
	attackerSigner := protoutil.SignerFromPrivate(*attacker)

	req := proto.CatchupBlocksRequest{
		Timestamp: protoutil.TimepointFromTime(time.Now()),
		Spans:     spans,
	}
	m := libxdr.NewXdrMarshaler()
	return proto.SignedCatchupBlocksRequest{
		Signer:    attackerSigner.Signer(),
		Signature: attackerSigner.SignXdr(m, &req),
		Request:   req,
	}
}

func newTestService(t *testing.T, maxBlocks int) *CatchupService {
	t.Helper()
	ctx := context.Background()
	rand := librand.NewRand()
	server := protoutil.MustGenerateIdentity(ctx, rand, "server:9999")
	serverSigner := protoutil.SignerFromPrivate(*server)
	pool := NewCatchupBlockPool(ctx, NewNoopCatchupPersistence())
	return NewCatchupService(ctx, serverSigner, libtime.NewTestTime(time.Now()), nil, pool, maxBlocks)
}

func TestCatchupSolicitRejectsOversizedSpan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := newTestService(t, 10)

	sreq := signedRequest(t, []proto.CatchupSpan{
		{From_block_height: 0, To_block_height: 100},
	})

	_, err := svc.Catchup_solicit(ctx, sreq)
	if err == nil {
		t.Fatal("expected error for oversized span, got nil")
	}
}

func TestCatchupSolicitAcceptsValidSpan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := newTestService(t, 10)

	sreq := signedRequest(t, []proto.CatchupSpan{
		{From_block_height: 0, To_block_height: 5},
	})

	_, err := svc.Catchup_solicit(ctx, sreq)
	if err != nil {
		t.Fatalf("expected success for valid span, got: %v", err)
	}
}

func TestCatchupSolicitReturnsErrorOnCanceledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := newTestService(t, 1000)

	sreq := signedRequest(t, []proto.CatchupSpan{
		{From_block_height: 0, To_block_height: 100},
	})

	_, err := svc.Catchup_solicit(ctx, sreq)
	if err == nil {
		t.Fatal("expected error on canceled context, got nil")
	}
}
