package dogwood

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/net/dogwood"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type DogwoodBenchmark struct {
	numPayBytes   int
	numIterations int
	cfg           proto.Config
	discovery     discovery.Discovery
	dogwood       *dogwood.DogwoodService
	wg            sync.WaitGroup
}

type DogwoodBenchmarkReport struct {
	NumPayBytes   int
	NumIterations int
	NumNodes      int
	Config        proto.DogwoodConfig
	//
	StatsMidway dogwood.Stats
	StatsEnd    dogwood.Stats
	//
	SendThroughput    libstat.CounterStats
	ReceiveThroughput libstat.CounterStats
	ReceptionRate     float64
}

func (x DogwoodBenchmarkReport) Summarize() DogwoodBenchmarkSummary {
	return DogwoodBenchmarkSummary{
		ResourceUsage:        x.StatsMidway.ResourceUsage,
		ReceptionLatency:     x.StatsMidway.TransmissionReceiverStats.Latency.Summarize(),
		ReceptionRate:        fmt.Sprintf("%.2f%%", x.ReceptionRate*100),
		SendThroughput:       humanize.Bytes(uint64(x.SendThroughput.SpeedPerSec)),
		ReceiveThroughput:    humanize.Bytes(uint64(x.ReceiveThroughput.SpeedPerSec)),
		Caller:               x.StatsMidway.CallerStats.Summarize(),
		StreamReceiver:       x.StatsMidway.StreamReceiverStats.Summarize(),
		StreamSender:         x.StatsMidway.StreamSenderStats.Summarize(),
		TransmissionReceiver: x.StatsMidway.TransmissionReceiverStats.Summarize(),
		TransmissionSender:   x.StatsMidway.TransmissionSenderStats.Summarize(),
		Navigator:            x.StatsMidway.NavigatorStats.Summarize(),
	}
}

type DogwoodBenchmarkSummary struct {
	ResourceUsage     string
	ReceptionLatency  string
	ReceptionRate     string
	SendThroughput    string
	ReceiveThroughput string
	//
	Caller               string
	StreamReceiver       string
	StreamSender         string
	TransmissionReceiver string
	TransmissionSender   string
	Navigator            string
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
		cfg:           cfg,
		discovery:     discovery,
		dogwood:       dw,
	}
}

func (bench *DogwoodBenchmark) selfString() string {
	return protoutil.VIDString(bench.discovery.SelfID())
}

func (bench *DogwoodBenchmark) Run(ctx context.Context, address string) {

	// start broadcast service
	bench.dogwood.Start(ctx, address, 3*time.Second)
	bench.dogwood.WaitReady(ctx)

	zeroPayload := make([]byte, bench.numPayBytes)

	var wg sync.WaitGroup

	numNodes := len(bench.cfg.Validator_config.Known_validators)

	// start benchmark sender

	var bytesSent libstat.Counter

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			liblog.Infof("%v sent %s, at %s/sec",
				bench.selfString(),
				humanize.Bytes(uint64(bytesSent.Sum())),
				humanize.Bytes(uint64(bytesSent.SpeedPerSec())))
		}()

		liblog.Infof("%v starting to send", bench.selfString())

		for j := 0; j < bench.numIterations; j++ {
			// if j%300 == 0 {
			// 	liblog.Infof("%v resource usage: %s", bench.selfString(), runtime.ResourceUsage())
			// }

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

	var bytesReceived libstat.Counter
	var transmissionsReceived int
	var statsMidway dogwood.Stats

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			liblog.Infof("%v received %.2f%% transmissions: %s at %s/sec",
				bench.selfString(),
				100*float64(transmissionsReceived)/float64(bench.numIterations*(numNodes-1)),
				humanize.Bytes(uint64(bytesReceived.Sum())),
				humanize.Bytes(uint64(bytesReceived.SpeedPerSec())))
		}()

		j := 0
		for {
			if j == (2*bench.numIterations)/3 {
				statsMidway = bench.dogwood.Stats()
			}
			// if j%300 == 0 {
			// 	liblog.Infof("%v resource usage: %s", bench.selfString(), runtime.ResourceUsage())
			// }

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
				transmissionsReceived += 1
			}

			j += 1
		}
	}()

	wg.Wait()

	// report

	report := DogwoodBenchmarkReport{
		NumPayBytes:   bench.numPayBytes,
		NumIterations: bench.numIterations,
		NumNodes:      numNodes,
		Config:        *bench.cfg.Dogwood_config,
		//
		StatsMidway:       statsMidway,
		StatsEnd:          bench.dogwood.Stats(),
		SendThroughput:    bytesSent.Stats(),
		ReceiveThroughput: bytesReceived.Stats(),
		ReceptionRate:     float64(transmissionsReceived) / float64(bench.numIterations*(numNodes-1)),
	}

	fmt.Println(form.Display(report))
	fmt.Println()
	fmt.Println(form.Display(report.Summarize()))
}

func (bench *DogwoodBenchmark) Start(ctx context.Context, address string) {
	bench.wg.Add(1)
	go func() {
		defer bench.wg.Done()
		bench.Run(ctx, address)
	}()
}

func (bench *DogwoodBenchmark) Wait() {
	bench.wg.Wait()
}
