package sched

import (
	"context"
	"net"

	"github.com/thruputchain/chuckd/util/netutil"
	"github.com/thruputchain/chuckd/util/rpcutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/librpc"
	proto "github.com/thruputchain/thruput-protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SchedulerCoordConn interface {
	proto.SchedulerCoord1
	Close()
}

type SchedulerCoordClient struct {
	proto.SchedulerCoord1_Client
	driver *rpc.Driver
}

func DialSchedulerCoordClient(ctx context.Context, dialer libnet.Dialer, address string) (SchedulerCoordConn, error) {
	return dialSchedulerClient(ctx, ctx, dialer, address)
}

func dialSchedulerClient(ctx, lifetime context.Context, dialer libnet.Dialer, address string) (SchedulerCoordConn, error) {
	var sock net.Conn
	var err error
	if isTCPDialer(dialer) {
		d := &net.Dialer{}
		d.SetMultipathTCP(false)
		sock, err = d.DialContext(ctx, "tcp", address)
	} else {
		sock, err = dialer.Dial(ctx, address)
	}
	if err != nil {
		return nil, err
	}
	return NewSchedulerCoordClient(lifetime, sock), nil
}

func NewSchedulerCoordClient(ctx context.Context, sock net.Conn) *SchedulerCoordClient {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &SchedulerCoordClient{
		SchedulerCoord1_Client: proto.SchedulerCoord1_Client{Send: driver, Ctx: ctx},
		driver:                 driver,
	}
}

func (cc *SchedulerCoordClient) Close() {
	cc.driver.Close()
}

func isTCPDialer(dialer libnet.Dialer) bool {
	switch dialer.(type) {
	case libnet.TCPNet, libnet.TCPReuseNet, *libnet.TCPNet, *libnet.TCPReuseNet:
		return true
	default:
		return false
	}
}

func dialProposalClient(ctx, lifetime context.Context, address string) (SchedulerCoordConn, error) {
	d := &net.Dialer{}
	d.SetMultipathTCP(true)
	sock, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	// Go negotiates TCP fallback for kernels and servers without MPTCP. A
	// diagnostic failure must not discard an otherwise working connection.
	active, probeErr := sock.(*net.TCPConn).MultipathTCP()
	if probeErr != nil {
		liblog.Infof(ctx, "SchedulerCoord proposal MPTCP status unavailable: %v", probeErr)
	} else {
		liblog.Infof(ctx, "SchedulerCoord proposal MPTCP active=%t", active)
	}
	return NewSchedulerCoordClient(lifetime, sock), nil
}
