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
	proto "github.com/groundhogchain/protocol-go"

	"github.com/dustin/go-humanize"
)

type BroadcastBenchmark struct {
	paySize       int
	payPerChannel int
	numIterations int
	listener      net.Listener
	broadcast     *broadcast.BroadcastService
}

func NewBroadcastBenchmark(
	ctx context.Context,
	rand *rand.Rand,
	outside libnet.Network,
	vcfg proto.ValidatorConfig,
	treeFanOut int, // fan out of multicast trees
	paySize int, // sife of message payload
	payPerChannel int, // number of message to send on a channel
	numIterations int,
) *BroadcastBenchmark {

	router := router.NewFixedRouter(ctx, vcfg.Private_info.Public_info, vcfg.Known_validators, treeFanOut)
	broadcast := broadcast.NewBroadcastService(
		ctx,
		proto.BroadcastConfig{
			Multicast_tree_ttl_msec: 3000,
			Chunk_size:              10e3,
		},
		outside,
		runtime.NewRandFromRoot(),
		vcfg.Private_info,
		router,
		broadcast.AcceptEverything{},
	)
	return &BroadcastBenchmark{
		paySize:       paySize,
		payPerChannel: payPerChannel,
		numIterations: numIterations,
		broadcast:     broadcast,
	}
}

func (bench *BroadcastBenchmark) Start(ctx context.Context, address string) {

	// start broadcast service
	bench.broadcast.Start(ctx, address)

	zeroPayload := make([]byte, bench.paySize)

	// start benchmark sender
	// XXX: needs speed control
	// XXX: is there backpressure on Send?
	go func() {
		var bytesSent libstat.Counter
		defer liblog.Infof("sent %s, at %s/sec",
			humanize.Bytes(uint64(bytesSent.Sum())),
			humanize.Bytes(uint64(bytesSent.SpeedPerSec())))

		for j := 0; j < bench.numIterations; j++ {
			s := bench.broadcast.StreamingAPI().MakeChannel(ctx)
			for i := 0; i < bench.payPerChannel; i++ {
				select {
				case <-ctx.Done(): // benchmark timeout
					return
				default:
				}

				// liblog.Infof("bench server %s sending message %d on channel %d",
				// 	protoutil.VIDString(bench.broadcast.ValidatorID()), i, j)

				sctx, cancel := context.WithTimeout(ctx, time.Minute) //XXX
				if err := s.Send(sctx, zeroPayload); err != nil {
					liblog.Errorf("bench send: %v", err)
				}
				cancel()

				bytesSent.Add(float64(len(zeroPayload)))
			}

			s.Close()
			liblog.Infof("sent %s, at %s/sec",
				humanize.Bytes(uint64(bytesSent.Sum())),
				humanize.Bytes(uint64(bytesSent.SpeedPerSec())))
		}
	}()

	// start benchmark receiver
	go func() {
		var bytesReceived libstat.Counter
		defer liblog.Infof("received %s, at %s/sec",
			humanize.Bytes(uint64(bytesReceived.Sum())),
			humanize.Bytes(uint64(bytesReceived.SpeedPerSec())))

		for {
			select {
			case <-ctx.Done(): // benchmark timeout
				return
			default:
			}

			r := bench.broadcast.StreamingAPI().AcceptChannel()

			go func() { // XXX: should we service them in sequence? does this block the service?
				defer r.Close()
				defer liblog.Infof("received %s, at %s/sec",
					humanize.Bytes(uint64(bytesReceived.Sum())),
					humanize.Bytes(uint64(bytesReceived.SpeedPerSec())))

				for {
					select {
					case <-ctx.Done(): // benchmark timeout
						return
					default:
					}
					rctx, cancel := context.WithTimeout(ctx, time.Minute) //XXX
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
