package coord

import (
	"context"
	"os"
	"testing"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/shard"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func TestMain(m *testing.M) {
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts()
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestNullRoundtrip(t *testing.T) {
	ctx := context.Background()

	addr := "localhost:8991"
	svc := NewCoordinatorService(ctx, libnet.TCPNet{}, runtime.NewRandFromRoot())
	svc.Start(ctx, addr)

	c, err := DialCoordinatorClientTCP(ctx, addr)
	must.NoError(ctx, err)
	c.Coordinator_null(ctx)
}

func TestShardCompleteness(t *testing.T) {
	ctx := context.Background()
	s := NewCoordinatorService(ctx, libnet.TCPNet{}, runtime.NewRandFromRoot())

	sh1 := shard.NewTestShardServer(ctx)
	sh2 := shard.NewTestShardServer(ctx)
	sh3 := shard.NewTestShardServer(ctx)
	sh1.Start(ctx)
	sh2.Start(ctx)
	sh3.Start(ctx)

	s.Coordinator_intro_shard(ctx, proto.ShardCard{
		Upid:         xdrutil.Uint64UPID(1),
		Block_height: 1,
		Shard_count:  2,
		Address:      sh1.Addr(),
	})

	if len(s.findCompleteDb()) > 0 {
		t.Fatalf("db should not be ready")
	}

	s.Coordinator_intro_shard(ctx, proto.ShardCard{
		Upid:         xdrutil.Uint64UPID(2),
		Block_height: 2, // different height
		Shard_count:  2,
		Address:      sh2.Addr(),
	})

	if len(s.findCompleteDb()) > 0 {
		t.Fatalf("db should not be ready")
	}

	s.Coordinator_intro_shard(ctx, proto.ShardCard{
		Upid:         xdrutil.Uint64UPID(3),
		Block_height: 1,
		Shard_count:  2,
		Address:      sh3.Addr(),
	})

	if len(s.findCompleteDb()) == 0 {
		t.Fatalf("db should be ready")
	}

}

func TestDbReadiness(t *testing.T) {

	ctx := context.Background()

	s := NewCoordinatorService(ctx, libnet.TCPNet{}, runtime.NewRandFromRoot())

	sh1 := shard.NewTestShardServer(ctx)
	sh2 := shard.NewTestShardServer(ctx)
	sh1.Start(ctx)
	sh2.Start(ctx)

	execUnblocked := make(chan int, 1)

	go func() {
		s.Coordinator_intro_exec(ctx, proto.ExecCard{
			Upid:    xdrutil.Uint64UPID(10),
			Address: "2.2.2.2",
		})
		execUnblocked <- 1
	}()

	s.Coordinator_intro_shard(ctx, proto.ShardCard{
		Upid:         xdrutil.Uint64UPID(1),
		Block_height: 1,
		Shard_count:  2,
		Address:      sh1.Addr(),
	})

	select {
	case <-execUnblocked:
		t.Errorf("executor should be blocked")
	default:
	}

	s.Coordinator_intro_shard(ctx, proto.ShardCard{
		Upid:         xdrutil.Uint64UPID(3),
		Block_height: 1,
		Shard_count:  2,
		Address:      sh2.Addr(),
	})

	<-execUnblocked
}
