package loadgen

import (
	"context"
	"net"
	"time"

	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type LoadgenClient struct {
	proto.LoadgenInterfaceV1_Client
	driver *rpc.Driver
}

func DialLoadgenClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*LoadgenClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewLoadgenClient(ctx, sock), nil
}

func DialLoadgenClientRetry(ctx context.Context, dialer libnet.AddressDialer, address string) (*LoadgenClient, error) {
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
