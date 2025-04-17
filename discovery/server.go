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
	vcfg proto.ValidatorConfig,
) *DiscoveryService {
	l, err := net.Listen("tcp", addr)
	must.NoError(ctx, err)
	return StartServer(ctx, cfg, libnet.TCPAddressDialer{}, l, vcfg)
}

func StartServer(
	ctx context.Context,
	cfg DiscoveryConfig,
	dialer libnet.AddressDialer,
	l net.Listener,
	vcfg proto.ValidatorConfig,
) *DiscoveryService {
	s := NewDiscoveryService(ctx, cfg, dialer, vcfg)
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
