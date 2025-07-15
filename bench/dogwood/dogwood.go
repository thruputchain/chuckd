package dogwood

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/router"
	proto "github.com/groundhogchain/protocol-go"
)

type DogwoodBenchmark struct {
	numPayBytes   int
	numIterations int
	dogwood       *dogwood.DogwoodService
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
		dogwood:       dw,
	}
}

func (bench *DogwoodBenchmark) Start(ctx context.Context, address string) {

	// start broadcast service
	bench.dogwood.Start(ctx, address)

	zeroPayload := make([]byte, bench.numPayBytes)

	// start benchmark sender
	go func() {

		time.Sleep(3 * time.Second) // XXX: wait for all nodes to warmup (flag???)
		liblog.Infof("starting to send")

		defer liblog.Infof("sender done")
		var bytesSent libstat.Counter

		for j := 0; j < bench.numIterations; j++ {

			err := bench.dogwood.Send(ctx, zeroPayload)
			if err != nil {
				liblog.Errorf("bench send: %v", err)
			} else {
				bytesSent.Add(float64(len(zeroPayload)))
			}

		}

		liblog.Infof("sent %s, at %s/sec",
			humanize.Bytes(uint64(bytesSent.Sum())),
			humanize.Bytes(uint64(bytesSent.SpeedPerSec())))
	}()

	// start benchmark receiver
	go func() {
		defer liblog.Infof("receiver done")
		var bytesReceived libstat.Counter

		defer func() {
			liblog.Infof("received %s, at %s/sec",
				humanize.Bytes(uint64(bytesReceived.Sum())),
				humanize.Bytes(uint64(bytesReceived.SpeedPerSec())))
		}()

		for {
			select {
			case <-ctx.Done(): // benchmark timeout
				return
			default:
			}
			rctx, cancel := context.WithTimeout(ctx, time.Second)
			payload, _, err := bench.dogwood.Receive(rctx)
			cancel()
			if errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if err != nil {
				liblog.Errorf("receive: %v", err)
			} else {
				bytesReceived.Add(float64(len(payload)))
			}
		}
	}()
}
