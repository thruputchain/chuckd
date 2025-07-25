package benchbroadcast

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/net/broadcast"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util"
	proto "github.com/groundhogchain/protocol-go"

	"github.com/dustin/go-humanize"
)

type BroadcastBenchmark struct {
	cfg              proto.BroadcastConfig
	chunksPerChannel int
	bytesPerSec      int
	numIterations    int
	listener         net.Listener
	broadcast        *broadcast.BroadcastService
}

func NewBroadcastBenchmark(
	ctx context.Context,
	rand *rand.Rand,
	outside libnet.Network,
	cfg proto.Config, // uses ValidatorConfig and BroadcastConfig
	fanOut int, // fan out of multicast trees
	chunksPerChannel int, // number of message to send on a channel
	bytesPerSec int,
	numIterations int,
) *BroadcastBenchmark {

	router := router.NewStaticUniformRouter(
		ctx,
		cfg.Validator_config.Private_info.Public_info,
		cfg.Validator_config.Known_validators,
		fanOut,
	)
	broadcast := broadcast.NewBroadcastService(
		ctx,
		*cfg.Broadcast_config,
		outside,
		runtime.NewRandFromRoot(),
		cfg.Validator_config.Private_info,
		router,
		broadcast.AcceptEverything{},
	)
	return &BroadcastBenchmark{
		cfg:              *cfg.Broadcast_config,
		chunksPerChannel: chunksPerChannel,
		bytesPerSec:      bytesPerSec,
		numIterations:    numIterations,
		broadcast:        broadcast,
	}
}

func (bench *BroadcastBenchmark) Start(ctx context.Context, address string) {

	// start broadcast service
	bench.broadcast.Start(ctx, address)

	zeroPayload := make([]byte, bench.cfg.Chunk_size)

	// start benchmark sender
	go func() {
		defer liblog.Infof("sender done")
		var bytesSent libstat.Counter

		chunksPerSec := util.CeilDivSigned(bench.bytesPerSec, int(bench.cfg.Chunk_size))
		liblog.Infof("sending %d chunks/sec", chunksPerSec)

		for j := 0; j < bench.numIterations; j++ {

			s := bench.broadcast.StreamingAPI().MakeChannel(ctx)

			Drip(
				ctx,
				chunksPerSec,
				bench.chunksPerChannel,
				func(i int) {
					if err := s.Send(ctx, zeroPayload); err != nil {
						liblog.Errorf("bench send: %v", err)
					}
					bytesSent.Add(float64(len(zeroPayload)))
				},
			)

			s.Close()

			liblog.Infof("sent %s, at %s/sec",
				humanize.Bytes(uint64(bytesSent.Sum())),
				humanize.Bytes(uint64(bytesSent.SpeedPerSec())))
		}
	}()

	// start benchmark receiver
	go func() {
		defer liblog.Infof("receiver done")
		var bytesReceived libstat.Counter

		for {
			select {
			case <-ctx.Done(): // benchmark timeout
				return
			default:
			}

			r := bench.broadcast.StreamingAPI().AcceptChannel()
			liblog.Infof("accepted channel")

			go func() {
				defer func() {
					liblog.Infof("received %s, at %s/sec",
						humanize.Bytes(uint64(bytesReceived.Sum())),
						humanize.Bytes(uint64(bytesReceived.SpeedPerSec())))
				}()
				defer r.Close()

				for {
					select {
					case <-ctx.Done(): // benchmark timeout
						return
					default:
					}
					rctx, cancel := context.WithTimeout(ctx, time.Second)
					if _, _, payload, err := r.Receive(rctx); err != nil {
						liblog.Errorf("receive: %v", err)
						cancel()
						return
					} else {
						bytesReceived.Add(float64(len(payload)))
					}
					cancel()
				}
			}()
		}
	}()
}

func (bench *BroadcastBenchmark) Close() {
	bench.listener.Close()
}
