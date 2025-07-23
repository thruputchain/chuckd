package echo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/xdrpp/goxdr/rpc"
)

type EchoBenchmark struct {
	net libnet.Network
	//
	numConns           int
	numWorkers         int
	numIterations      int
	numBytesPerPayload int
	//
	clients []*echoClient
	workers libflow.InvokeWorkerPool[callEchoArg]
	//
	numSent     libstat.Counter
	numReceived libstat.Counter
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
	maxPendingCalls int,
	numIterations int,
	numBytesPerPayload int,
) *EchoBenchmark {
	x := &EchoBenchmark{
		net:                net,
		numConns:           numConns,
		numWorkers:         maxPendingCalls,
		numIterations:      numIterations,
		numBytesPerPayload: numBytesPerPayload,
	}
	x.workers = libflow.NewInvokeWorkerPool(ctx, x.callEcho, maxPendingCalls)
	return x
}

func (x *EchoBenchmark) Report() string {
	return fmt.Sprintf("sent %.1f req/sec, received %.1f req/sec", x.numSent.SpeedPerSec(), x.numReceived.SpeedPerSec())
}

func (x *EchoBenchmark) Run(ctx context.Context, ourAddress string, theirAddress string) {
	rpcutil.Start(ctx, x.net, ourAddress, Echo1_Server{Srv: x})

	// all the other server grace time to start
	time.Sleep(3 * time.Second)

	x.runSender(ctx, theirAddress)

	fmt.Println(x.Report())
}

// receiver

func (x *EchoBenchmark) Null(context.Context) error {
	return nil
}

func (x *EchoBenchmark) Echo(_ context.Context, req Request) (Reply, error) {
	x.numReceived.Add(1)
	return Reply{Payload: req.Payload}, nil
}

// sender

func (x *EchoBenchmark) runSender(ctx context.Context, address string) {

	// create connections
	x.clients = make([]*echoClient, x.numConns)
	for i := range x.clients {
		c, err := DialEchoClient(ctx, x.net, rpc.NewMsgPool(), address)
		must.NoError(ctx, err)
		x.clients[i] = &echoClient{EchoClient: c}
		defer c.Close()
	}

	payload := make([]byte, x.numBytesPerPayload)

	// workload
	var wg sync.WaitGroup
	for i := 0; i < x.numIterations; i++ {
		wg.Add(1)
		x.workers.Invoke(callEchoArg{wg: &wg, ctx: ctx, payload: payload})
	}
	wg.Wait()

}

type callEchoArg struct {
	wg      *sync.WaitGroup
	ctx     context.Context
	payload []byte
}

func (x *EchoBenchmark) callEcho(arg callEchoArg) {
	defer arg.wg.Done()

	c := x.pickClient()
	req := Request{Payload: arg.payload}

	c.addNumPending(1)
	_, err := c.Echo(arg.ctx, req)
	must.NoError(arg.ctx, err)
	c.addNumPending(-1)

	x.numSent.Add(1)
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
		panic("negative pending count")
	}
}
