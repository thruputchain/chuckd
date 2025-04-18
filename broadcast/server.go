package broadcast

import (
	"context"
	"math/rand"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/router"
	"github.com/groundhogchain/chuckd/test"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

func StartServerTCP(
	ctx context.Context,
	cfg BroadcastConfig,
	rr *rand.Rand,
	addr string,
	privInfo proto.ValidatorPrivateInfo,
	router router.Router,
	disc AcceptDiscriminator,
) *BroadcastService {
	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)
	return StartServer(ctx, cfg, libnet.TCPAddressDialer{}, rr, l, privInfo, router, disc)
}

func StartServerTest(
	ctx context.Context,
	cfg BroadcastConfig,
	rr *rand.Rand,
	testnet *test.TestNetwork,
	addr string,
	privInfo proto.ValidatorPrivateInfo,
	router router.Router,
	disc AcceptDiscriminator,
) *BroadcastService {
	l, err := testnet.Listen("", addr)
	must.NoError(ctx, err)
	return StartServer(ctx, cfg, testnet, rr, l, privInfo, router, disc)
}

func StartServer(
	ctx context.Context,
	cfg BroadcastConfig,
	dialer libnet.AddressDialer,
	rr *rand.Rand,
	l net.Listener,
	privInfo proto.ValidatorPrivateInfo,
	router router.Router,
	disc AcceptDiscriminator,
) *BroadcastService {
	svc := NewBroadcastService(ctx, cfg, dialer, rr, privInfo, router, disc)
	go func() {
		defer l.Close()
		for {
			sock, err := l.Accept()
			must.NoError(ctx, err)
			driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
			driver.Lock = nil // process all calls in parallel
			// driver.Log = log.Writer()
			driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
			driver.Register(proto.Broadcast1_Server{Srv: svc})
			go driver.Go()
		}
	}()
	return svc
}
