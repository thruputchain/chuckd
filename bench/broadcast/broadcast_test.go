package benchbroadcast

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	liblog.LogLevel(zerolog.DebugLevel)
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts(true)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestBroadcast(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	testTreeFanOut := 5
	testChunksPerChannel := 10
	testBytesPerSec := 10000
	testIterations := 10

	rr := runtime.NewRandFromRoot()
	r0 := runtime.NewRandFromRoot()
	r1 := runtime.NewRandFromRoot()
	r2 := runtime.NewRandFromRoot()

	a0 := "127.0.0.1:22000"
	a1 := "127.0.0.1:22001"
	a2 := "127.0.0.1:22002"

	priv0 := protoutil.MustGenerateIdentity(ctx, rr, a0)
	priv1 := protoutil.MustGenerateIdentity(ctx, rr, a1)
	priv2 := protoutil.MustGenerateIdentity(ctx, rr, a2)

	known := []proto.ValidatorPublicInfo{priv0.Public_info, priv1.Public_info, priv2.Public_info}

	bcfg := &proto.BroadcastConfig{
		Multicast_tree_ttl_msec: 1000 * 3,
		Chunk_size:              1000,
	}
	cfg0 := proto.Config{
		Validator_config: &proto.ValidatorConfig{Private_info: *priv0, Known_validators: known},
		Broadcast_config: bcfg,
	}
	cfg1 := proto.Config{
		Validator_config: &proto.ValidatorConfig{Private_info: *priv1, Known_validators: known},
		Broadcast_config: bcfg,
	}
	cfg2 := proto.Config{
		Validator_config: &proto.ValidatorConfig{Private_info: *priv2, Known_validators: known},
		Broadcast_config: bcfg,
	}

	n := libnet.TCPNet{}
	bb0 := NewBroadcastBenchmark(ctx, r0, n, cfg0, testTreeFanOut, testChunksPerChannel, testBytesPerSec, testIterations)
	bb1 := NewBroadcastBenchmark(ctx, r1, n, cfg1, testTreeFanOut, testChunksPerChannel, testBytesPerSec, testIterations)
	bb2 := NewBroadcastBenchmark(ctx, r2, n, cfg2, testTreeFanOut, testChunksPerChannel, testBytesPerSec, testIterations)

	bb0.Start(ctx, a0)
	bb1.Start(ctx, a1)
	bb2.Start(ctx, a2)

	<-ctx.Done()
}
