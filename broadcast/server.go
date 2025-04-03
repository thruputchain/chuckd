package broadcast

import (
	"context"
	"log"
	"net"

	"github.com/groundhogchain/chuckd/discovery"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

func StartServer(
	ctx context.Context,
	addr string,
	privInfo proto.ValidatorPrivateInfo,
	ourSlices []proto.QuorumSlice,
	discovery *discovery.DiscoveryService,
) {
	go func() {
		s := NewBroadcastService(ctx, privInfo, ourSlices, discovery)

		l, err := net.Listen("tcp", addr)
		must.NoError(ctx, err)
		defer l.Close()

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
