package echo

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/xdrpp/goxdr/rpc"
)

type EchoBenchmark struct {
	net                libnet.Network
	numConns           int
	numWorkers         int
	numIterations      int
	numBytesPerPayload int
	clients            []*echoClient
	workers            libflow.SubmitWorkerPool
}

type echoClient struct {
	*EchoClient
	lk       sync.Mutex
	npending int
}

func NewEchoBench(
	ctx context.Context,
	net libnet.Network,
	numConns int,
	numWorkers int,
	numIterations int,
	numBytesPerPayload int,
) *EchoBenchmark {
	return &EchoBenchmark{
		net:                net,
		numConns:           numConns,
		numWorkers:         numWorkers,
		numIterations:      numIterations,
		numBytesPerPayload: numBytesPerPayload,
		workers:            libflow.NewSubmitWorkerPool(ctx, numWorkers),
	}
}

func (x *EchoBenchmark) Start(ctx context.Context, ourAddress string, theirAddress string) {

}

func (x *EchoBenchmark) runCall(ctx context.Context, address string) {

	// wait for server to start
	time.Sleep(3 * time.Second)

	// create connections
	x.clients = make([]*echoClient, x.numConns)
	for i := range x.clients {
		c, err := DialEchoClient(ctx, x.net, rpc.NewMsgPool(), address)
		must.NoError(ctx, err)
		x.clients[i] = &echoClient{EchoClient: c}
	}

	payload := make([]byte, x.numBytesPerPayload)

	// workload
	for i := 0; i < x.numIterations; i++ {
		x.workers.Submit(
			func() {
				c := x.pickClient()

				req := Request{Payload: payload}
				_, err := c.Echo(ctx, req)
				must.NoError(ctx, err)

				// XXX // record
			},
		)
	}
}

func (x *EchoBenchmark) pickClient() *echoClient {

	var minClient *echoClient
	var minPending int

	for _, c := range x.clients {
		npending := c.getNumPending()
		if minClient == nil || minPending > npending {
			minClient, minPending = c, npending
		}
	}

	return minClient
}

func (x *echoClient) getNumPending() int {
	x.lk.Lock()
	defer x.lk.Unlock()
	return x.npending
}

func (x *echoClient) addNumPending(delta int) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.npending += delta
	if x.npending < 0 {
		panic("negative pending conut")
	}
}
