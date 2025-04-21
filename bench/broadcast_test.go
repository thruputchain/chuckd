package bench

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func TestMain(m *testing.M) {
	log.LogVerbosely()
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts()
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestBroadcast(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	testDeliveryPar := 5
	testTreeFanOut := 5
	testPaySize := 10
	testPayPerChannel := 10
	testIterations := 1

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

	vcfg0 := proto.ValidatorConfig{Private_info: *priv0, Known_validators: known}
	vcfg1 := proto.ValidatorConfig{Private_info: *priv1, Known_validators: known}
	vcfg2 := proto.ValidatorConfig{Private_info: *priv2, Known_validators: known}

	bb0 := NewBroadcastBenchmark(ctx, r0, vcfg0, testDeliveryPar, testTreeFanOut, testPaySize, testPayPerChannel, testIterations)
	bb1 := NewBroadcastBenchmark(ctx, r1, vcfg1, testDeliveryPar, testTreeFanOut, testPaySize, testPayPerChannel, testIterations)
	bb2 := NewBroadcastBenchmark(ctx, r2, vcfg2, testDeliveryPar, testTreeFanOut, testPaySize, testPayPerChannel, testIterations)

	bb0.Start(ctx, a0)
	bb1.Start(ctx, a1)
	bb2.Start(ctx, a2)

	<-ctx.Done()
}
