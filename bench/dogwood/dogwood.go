package dogwood

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type DogwoodBenchmark struct {
	numPayBytes   int
	numIterations int
	discovery     discovery.Discovery
	dogwood       *dogwood.DogwoodService
	wg            sync.WaitGroup
}

func NewDogwoodBenchmark(
	ctx context.Context,
	rand *rand.Rand,
	net libnet.Network,
	cfg proto.Config, // uses ValidatorConfig and BroadcastConfig
	fanOut int,
	numPayBytes int,
	numIterations int,
) *DogwoodBenchmark {

	discovery := discovery.NewStaticDiscovery(
		cfg.Validator_config.Private_info,
		cfg.Validator_config.Known_validators,
	)

	router := router.NewFixedRouter(
		ctx,
		cfg.Validator_config.Private_info.Public_info,
		cfg.Validator_config.Known_validators,
		fanOut,
	)

	dw := dogwood.NewDogwoodService(
		ctx,
		*cfg.Dogwood_config,
		libtime.RealTime{},
		rand,
		net,
		discovery,
		router,
	)
	return &DogwoodBenchmark{
		numPayBytes:   numPayBytes,
		numIterations: numIterations,
		discovery:     discovery,
		dogwood:       dw,
	}
}

func (bench *DogwoodBenchmark) selfString() string {
	return protoutil.VIDString(bench.discovery.SelfID())
}

func (bench *DogwoodBenchmark) Start(ctx context.Context, address string) {

	// start broadcast service
	bench.dogwood.Start(ctx, address)

	zeroPayload := make([]byte, bench.numPayBytes)

	// start benchmark sender
	bench.wg.Add(1)
	go func() {
		defer bench.wg.Done()

		defer liblog.Infof("%v sender done", bench.selfString())
		var bytesSent libstat.Counter

		defer func() {
			liblog.Infof("%v sent %s, at %s/sec",
				bench.selfString(),
				humanize.Bytes(uint64(bytesSent.Sum())),
				humanize.Bytes(uint64(bytesSent.SpeedPerSec())))
		}()

		time.Sleep(3 * time.Second) // XXX: wait for all nodes to warmup (flag???)
		liblog.Infof("%v starting to send", bench.selfString())

		for j := 0; j < bench.numIterations; j++ {
			if j%10 == 0 {
				liblog.Infof("resource usage: %s", runtime.ResourceUsage())
			}

			select {
			case <-ctx.Done():
				return
			default:
			}

			err := bench.dogwood.Send(ctx, zeroPayload)
			if err != nil {
				liblog.Errorf("%v bench send: %v", bench.selfString(), err)
			} else {
				bytesSent.Add(float64(len(zeroPayload)))
			}

		}

	}()

	// start benchmark receiver
	bench.wg.Add(1)
	go func() {
		defer bench.wg.Done()

		defer liblog.Infof("%v receiver done", bench.selfString())
		var bytesReceived libstat.Counter

		defer func() {
			liblog.Infof("%v received %s, at %s/sec",
				bench.selfString(),
				humanize.Bytes(uint64(bytesReceived.Sum())),
				humanize.Bytes(uint64(bytesReceived.SpeedPerSec())))
		}()

		j := 0
		for {
			if j%10 == 0 {
				liblog.Infof("resource usage: %s", runtime.ResourceUsage())
			}

			select {
			case <-ctx.Done(): // benchmark timeout
				return
			default:
			}

			rctx, cancel := context.WithTimeout(ctx, 5*time.Second) // absorb 3 sec warm up + if nothing comes for more than a second, end
			payload, _, err := bench.dogwood.Receive(rctx)
			cancel()
			// payload, _, err := bench.dogwood.Receive(ctx)
			if errors.Is(err, context.DeadlineExceeded) {
				liblog.Errorf("%v receive: %v", bench.selfString(), err)
				return
			}
			if err != nil {
				liblog.Errorf("%v receive: %v", bench.selfString(), err)
			} else {
				bytesReceived.Add(float64(len(payload)))
			}

			j += 1
		}
	}()
}

func (bench *DogwoodBenchmark) Wait() {
	bench.wg.Wait()
}
