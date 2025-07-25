package dogwood

import (
	"context"
	"os"
	"testing"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	liblog.LogLevel(zerolog.InfoLevel)
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts(true)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestDogwood(t *testing.T) {

	ctx := context.Background()
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	// defer cancel()

	testFanOut := 5
	testBytesPerPay := 300000
	testNumIterations := 50000

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

	dcfg := &proto.DogwoodConfig{
		Route_count:             7,
		Route_time_to_live_msec: 60e3, // 1 minute
		Route_time_to_use_msec:  200,  // 200 msec
		//
		Endpoint_max_pending:       100,
		Endpoint_num_clients:       10,
		Endpoint_idle_ttl_msec:     60e3, // 1 minute
		Endpoint_redial_delay_msec: 50,   // 50 msec
		//
		Stream_receiver_worker_count: 10,
		Stream_sender_worker_count:   2,
		//
		Transmission_receiver_decode_worker_count: 10,
		Transmission_sender_worker_count:          2,
		//
		Transmission_partial_max_ttl_msec:  10e3, // 10 sec
		Transmission_partial_idle_ttl_msec: 1e3,  // 1 sec
	}
	cfg0 := proto.Config{
		Validator_config: &proto.ValidatorConfig{Private_info: *priv0, Known_validators: known},
		Dogwood_config:   dcfg,
	}
	cfg1 := proto.Config{
		Validator_config: &proto.ValidatorConfig{Private_info: *priv1, Known_validators: known},
		Dogwood_config:   dcfg,
	}
	cfg2 := proto.Config{
		Validator_config: &proto.ValidatorConfig{Private_info: *priv2, Known_validators: known},
		Dogwood_config:   dcfg,
	}

	n := libnet.TCPNet{}
	// n := quic.NewQUICNet(ctx)
	bb0 := NewDogwoodBenchmark(ctx, r0, n, cfg0, testFanOut, testBytesPerPay, testNumIterations)
	bb1 := NewDogwoodBenchmark(ctx, r1, n, cfg1, testFanOut, testBytesPerPay, testNumIterations)
	bb2 := NewDogwoodBenchmark(ctx, r2, n, cfg2, testFanOut, testBytesPerPay, testNumIterations)

	bb0.Start(ctx, a0)
	bb1.Start(ctx, a1)
	bb2.Start(ctx, a2)

	bb0.Wait()
	bb1.Wait()
	bb2.Wait()
	// <-ctx.Done()
}
