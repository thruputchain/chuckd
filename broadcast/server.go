package broadcast

import (
	"context"
	"log"
	"math/rand"
	"net"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/test"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

func StartServerTCP(
	ctx context.Context,
	rr *rand.Rand,
	addr string,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	router router.Router,
	disc AcceptDiscriminator,
) {
	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)
	StartServer(ctx, rr, l, privInfo, ourSlices, router, disc)
}

func StartServerTest(
	ctx context.Context,
	rr *rand.Rand,
	vnet *test.TestNetwork,
	addr string,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	router router.Router,
	disc AcceptDiscriminator,
) {
	l, err := vnet.Listen("", addr)
	must.NoError(ctx, err)
	StartServer(ctx, rr, l, privInfo, ourSlices, router, disc)
}

func StartServer(
	ctx context.Context,
	rr *rand.Rand,
	l net.Listener,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	router router.Router,
	disc AcceptDiscriminator,
) {
	go func() {
		defer l.Close()
		s := NewBroadcastService(ctx, rr, privInfo, ourSlices, router, disc)
		for {
			sock, err := l.Accept()
			must.NoError(ctx, err)
			driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
			driver.Lock = nil // process all calls in parallel
			driver.Log = log.Writer()
			driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
			driver.Register(proto.Broadcast1_Server{Srv: s})
			go driver.Go()
		}
	}()
}
