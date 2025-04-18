package bench

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/groundhogchain/chuckd/broadcast"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/runtime"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type BroadcastBenchmark struct {
	paySize       int
	payPerChannel int
	listener      net.Listener
	broadcast     *broadcast.BroadcastService
}

func NewBroadcastBenchmark(
	ctx context.Context,
	rr *rand.Rand,
	addr string,
	vcfg proto.ValidatorConfig,
	deliveryPar int,
	treeFanOut int, // fanout of multicast trees
	paySize int, // sife of message payload
	payPerChannel int, // number of message to send on a channel
) *BroadcastBenchmark {

	log.Infof("starting broadcast service on %s", addr)
	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)

	router := router.NewFixedRouter(ctx, vcfg.Private_info.Public_info, vcfg.Known_validators, treeFanOut)
	broadcast := broadcast.NewBroadcastService(
		ctx,
		broadcast.BroadcastConfig{
			DeliveryPar:            deliveryPar,
			UnusedMulticastTreeTTL: time.Minute,
		},
		libnet.TCPAddressDialer{},
		runtime.NewRandFromRoot(),
		vcfg.Private_info,
		router,
		broadcast.AcceptEverything{},
	)
	return &BroadcastBenchmark{
		paySize:       paySize,
		payPerChannel: payPerChannel,
		listener:      l,
		broadcast:     broadcast,
	}
}

func (bench *BroadcastBenchmark) Start(ctx context.Context) {

	// start broadcast service
	go func() {
		sock, err := bench.listener.Accept()
		must.NoError(ctx, err)
		driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
		driver.Lock = nil // process all calls in parallel
		// driver.Log = log.Writer()
		driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
		driver.Register(proto.Broadcast1_Server{Srv: bench.broadcast})
		go driver.Go()
		<-ctx.Done()
	}()

	// start benchmark sender
	go func() {
		for {
			s := bench.broadcast.API().MakeChannel()
		sendLoop:
			for i := 0; i < bench.payPerChannel; i++ {
				select {
				case <-ctx.Done(): // benchmark timeout
					break sendLoop
				default:
				}
				sctx, cancel := context.WithTimeout(ctx, time.Minute) //XXX
				if err := s.Send(sctx, make([]byte, bench.paySize)); err != nil {
					log.Errorf("send: %v", err)
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
			r := bench.broadcast.API().AcceptChannel()
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
