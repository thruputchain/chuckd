package coord

import (
	"context"
	"testing"

	"github.com/groundhogchain/chuckd/xdr/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func TestNullRoundtrip(t *testing.T) {
	ctx := context.Background()

	addr := "localhost:8991"
	StartServer(ctx, addr)

	c := NewCoordinatorClient(ctx, addr)
	c.Coordinator_null()
}

func TestDbReadiness(t *testing.T) {
	ctx := context.Background()
	s := NewCoordinatorService(ctx)

	s.Coordinator_intro_db(proto.DbCard{
		Upid:         xdrutil.Uint64UPID(1),
		Block_height: 1,
		Shard_count:  2,
		Address:      "1.1.1.1",
	})

	if s.Coordinator_list_db().Is_ready {
		t.Fatalf("db should not be ready")
	}

	s.Coordinator_intro_db(proto.DbCard{
		Upid:         xdrutil.Uint64UPID(2),
		Block_height: 2, // different height
		Shard_count:  2,
		Address:      "1.1.1.2",
	})

	if s.Coordinator_list_db().Is_ready {
		t.Fatalf("db should not be ready")
	}

	s.Coordinator_intro_db(proto.DbCard{
		Upid:         xdrutil.Uint64UPID(3),
		Block_height: 1,
		Shard_count:  2,
		Address:      "1.1.1.3",
	})

	if !s.Coordinator_list_db().Is_ready {
		t.Fatalf("db should be ready")
	}

}
