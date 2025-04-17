package coord

import (
	"context"
	"log"
	"math/rand"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

func StartServerTCP(
	ctx context.Context,
	rr *rand.Rand,
	address string,
) *CoordinatorService {
	l, err := net.Listen("tcp", address)
	must.NoError(ctx, err)
	return StartServer(ctx, rr, libnet.TCPAddressDialer{}, l)
}

func StartServer(
	ctx context.Context,
	rr *rand.Rand,
	dialer libnet.AddressDialer,
	l net.Listener,
) *CoordinatorService {
	svc := NewCoordinatorService(ctx, dialer, rr)
	go func() {
		defer l.Close()
		for {
			sock, err := l.Accept()
			must.NoError(ctx, err)
			driver := rpc.NewDriver(ctx, rpc.NewStreamTransport(sock))
			driver.Lock = nil // process all calls in parallel
			driver.Log = log.Writer()
			driver.PanicHandler = rpc.PanicHandlerFunc(must.LogPanic)
			driver.Register(proto.Coordinator1_Server{Srv: svc})
			go driver.Go()
		}
	}()
	return svc
}
