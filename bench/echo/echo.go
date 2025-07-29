package echo

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libflow"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/xdrpp/goxdr/rpc"
)

type EchoBenchmark struct {
	net libnet.Network
	//
	sendStartDelay time.Duration
	receiveTimeout time.Duration
	//
	doSend             bool
	numConns           int
	numMaxPendingCalls int
	numIterations      int
	numBytesPerPayload int
	noEcho             bool
	//
	clients        []*echoClient
	workers        libflow.InvokeWorkerPool[callEchoArg]
	abortSend      sync.Once
	sendAborted    chan struct{}
	abortReceive   sync.Once
	receiveAborted chan struct{}
	//
	lk           sync.Mutex
	lastReceived time.Time
	//
	numSent     libstat.Counter
	numReceived libstat.Counter
	roundtrip   libstat.LatencyMeter
}

type echoClient struct {
	*EchoClient
	lk       sync.Mutex
	npending int
}

func NewEchoBench(
	ctx context.Context,
	net libnet.Network,
	doSend bool,
	numConns int,
	maxPendingCalls int,
	numIterations int,
	numBytesPerPayload int,
	noEcho bool,
) *EchoBenchmark {
	x := &EchoBenchmark{
		net:                net,
		sendStartDelay:     3 * time.Second,
		receiveTimeout:     5 * time.Second, // must be larger than sendStartDelay
		doSend:             doSend,
		numConns:           numConns,
		numMaxPendingCalls: maxPendingCalls,
		numIterations:      numIterations,
		numBytesPerPayload: numBytesPerPayload,
		noEcho:             noEcho,
		sendAborted:        make(chan struct{}),
		receiveAborted:     make(chan struct{}),
	}
	x.workers = libflow.NewInvokeWorkerPool(ctx, x.callEcho, maxPendingCalls*numConns)
	return x
}

func (x *EchoBenchmark) Report() string {
	var w bytes.Buffer
	fmt.Fprintf(&w, "do-send=%v\n", x.doSend)
	fmt.Fprintf(&w, "num-conns=%d, max-pending-per-conn=%d, num-iterations=%d, bytes-per-pay=%d, no-echo=%v\n",
		x.numConns, x.numMaxPendingCalls, x.numIterations, x.numBytesPerPayload, x.noEcho)
	fmt.Fprintln(&w, "---")
	fmt.Fprintf(&w, "sent %d reqs, received %d reqs\n", x.numSent.Count(), x.numReceived.Count())
	fmt.Fprintf(&w, "sent at %.1f req/sec, received at %.1f req/sec\n", x.numSent.SpeedPerSec(), x.numReceived.SpeedPerSec())
	fmt.Fprintf(&w, "roundtrip %s\n", x.roundtrip.String())
	fmt.Fprintf(&w, "workers %s\n", x.workers.Stats().Summarize())
	return w.String()
}

func (x *EchoBenchmark) Run(ctx context.Context, ourAddress string, theirAddress string) {

	if x.doSend {

		// all the other server grace time to start
		time.Sleep(x.sendStartDelay)

		x.do(ctx, theirAddress)

	} else {

		x.updateReceptionTime()
		rpcutil.StartSerial(ctx, x.net, ourAddress, Echo1_Server{Srv: x})

		// watch for no activity
		go func() {
			for {
				time.Sleep(x.receiveTimeout)
				if x.noReceptionActivity() {
					x.abortReceive.Do(func() { close(x.receiveAborted) })
					return
				}
			}
		}()

		<-x.receiveAborted

	}

	fmt.Println(x.Report())
}

// receiver

func (x *EchoBenchmark) updateReceptionTime() {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.lastReceived = time.Now()
}

func (x *EchoBenchmark) noReceptionActivity() bool {
	x.lk.Lock()
	defer x.lk.Unlock()
	return time.Since(x.lastReceived) >= x.receiveTimeout
}

func (x *EchoBenchmark) Null(context.Context) error {
	x.updateReceptionTime()
	x.numReceived.Add(1)
	return nil
}

func (x *EchoBenchmark) Echo(_ context.Context, req Request) (Reply, error) {
	x.updateReceptionTime()
	x.numReceived.Add(1)
	return Reply{Payload: req.Payload}, nil
}

func (x *EchoBenchmark) No_echo(_ context.Context, req Request) error {
	x.updateReceptionTime()
	x.numReceived.Add(1)
	return nil
}

// sender

func (x *EchoBenchmark) do(ctx context.Context, address string) {

	// create connections
	x.clients = make([]*echoClient, x.numConns)
	for i := range x.clients {
		c, err := DialEchoClient(ctx, x.net, rpc.NewMessagePool(), address)
		must.NoError(ctx, err)
		x.clients[i] = &echoClient{EchoClient: c}
		defer c.Close()
	}

	payload := make([]byte, x.numBytesPerPayload)

	var wg sync.WaitGroup
workload:
	for i := 0; i < x.numIterations; i++ {
		select {
		case <-x.sendAborted:
			break workload
		default:
		}

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

	select {
	case <-x.sendAborted:
		return
	default:
	}

	c := x.pickClient()
	req := Request{Payload: arg.payload}

	var err error

	ctx, cancel := context.WithTimeout(arg.ctx, time.Second)

	c.addNumPending(1)
	span := x.roundtrip.Start()
	if x.noEcho {
		err = c.No_echo(ctx, req)
	} else {
		_, err = c.Echo(ctx, req)
	}
	cancel()
	span.Stop()
	c.addNumPending(-1)

	if err != nil {
		liblog.Errorf("call: %v", err)
		x.abortSend.Do(func() { close(x.sendAborted) })
		return
	}

	x.numSent.Add(1)
}

func (x *EchoBenchmark) pickClient() *echoClient {

	var minClient *echoClient
	var minPending int

	perm := rand.Perm(len(x.clients))
	for i := range x.clients {
		c := x.clients[perm[i]]
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
