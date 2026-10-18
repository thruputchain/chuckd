package loadgen

import (
	"context"
	"net"
	"time"

	"github.com/thruputchain/chuckd/util/netutil"
	"github.com/thruputchain/chuckd/util/rpcutil"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/librpc"
	proto "github.com/thruputchain/thruput-protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type LoadgenClient struct {
	proto.LoadgenInterfaceV1_Client
	driver *rpc.Driver
}

func DialLoadgenClient(ctx context.Context, dialer libnet.Dialer, address string) (*LoadgenClient, error) {
	sock, err := dialer.Dial(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewLoadgenClient(ctx, sock), nil
}

func DialLoadgenClientRetry(ctx context.Context, dialer libnet.Dialer, address string) (*LoadgenClient, error) {
	for {
		client, err := DialLoadgenClient(ctx, dialer, address)
		if err == nil {
			return client, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func NewLoadgenClient(ctx context.Context, sock net.Conn) *LoadgenClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &LoadgenClient{
		LoadgenInterfaceV1_Client: proto.LoadgenInterfaceV1_Client{Send: driver, Ctx: ctx},
		driver:                    driver,
	}
}

func (cc *LoadgenClient) Close() {
	cc.driver.Close()
}
