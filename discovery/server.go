package discovery

import (
	"context"
	"log"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

func StartServerTCP(
	ctx context.Context,
	cfg DiscoveryConfig,
	addr string,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
) *DiscoveryService {
	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)
	return StartServer(ctx, cfg, libnet.TCPAddressDialer{}, l, privInfo, ourSlices)
}

func StartServer(
	ctx context.Context,
	cfg DiscoveryConfig,
	dialer libnet.AddressDialer,
	l net.Listener,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
) *DiscoveryService {
	s := NewDiscoveryService(ctx, cfg, dialer, privInfo, ourSlices)
	go func() {
		defer l.Close()

		for {
			sock, err := l.Accept()
			must.NoError(ctx, err)
			driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
			driver.Lock = nil // process all calls in parallel
			driver.Log = log.Writer()
			driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
			driver.Register(proto.Discovery1_Server{Srv: s})
			go driver.Go()
		}
	}()
	return s
}
