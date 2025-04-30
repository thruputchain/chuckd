package bench

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/groundhogchain/chuckd/broadcast"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
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
	rr *rand.Rand,
	vcfg proto.ValidatorConfig,
	treeFanOut int, // fanout of multicast trees
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
		libnet.TCPNet{},
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

	// start benchmark sender
	go func() {
		for j := 0; j < bench.numIterations; j++ {
			s := bench.broadcast.StreamingAPI().MakeChannel(ctx)
		sendLoop:
			for i := 0; i < bench.payPerChannel; i++ {
				select {
				case <-ctx.Done(): // benchmark timeout
					break sendLoop
				default:
				}
				sctx, cancel := context.WithTimeout(ctx, time.Minute) //XXX
				log.Infof("bench server %s sending message %d on channel %d", protoutil.ValidatorIDString(bench.broadcast.ValidatorID()), i, j)
				if err := s.Send(sctx, make([]byte, bench.paySize)); err != nil {
					log.Errorf("bench send: %v", err)
				}
				cancel()
			}
			s.Close()
		}
	}()

	// start benchmark receiver
	go func() {
	acceptLoop:
		for {
			select {
			case <-ctx.Done(): // benchmark timeout
				break acceptLoop
			default:
			}
			r := bench.broadcast.StreamingAPI().AcceptChannel()
			go func() {
				defer r.Close()
			receiveLoop:
				for {
					select {
					case <-ctx.Done(): // benchmark timeout
						break receiveLoop
					default:
					}
					rctx, cancel := context.WithTimeout(ctx, time.Minute) //XXX
					if _, _, _, err := r.Receive(rctx); err != nil {
						log.Errorf("receive: %v", err)
						cancel()
						break receiveLoop
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
