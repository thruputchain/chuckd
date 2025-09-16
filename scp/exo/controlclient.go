package exo

import (
	"context"
	"net"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SCPControlClient struct {
	proto.SCP_Control1_Client
	driver *rpc.Driver
}

func DialSCPControlClient(ctx context.Context, dialer libnet.AddressDialer, address string) (*SCPControlClient, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewSCPControlClient(ctx, sock), nil
}

func NewSCPControlClient(ctx context.Context, sock net.Conn) *SCPControlClient {
	driver := rpcutil.NewDriverSerial(ctx, sock)
	go driver.Go()

	return &SCPControlClient{
		SCP_Control1_Client: proto.SCP_Control1_Client{Send: driver, Ctx: ctx},
		driver:              driver,
	}
}

func (cc *SCPControlClient) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	err := cc.Scp_control_received_envelope(ctx, env)
	must.NoError(ctx, err)
}

func (cc *SCPControlClient) Nominate(ctx context.Context, slot scpapi.Slot, value scpapi.Value, prev scpapi.Value) bool {
	r, err := cc.Scp_control_nominate(ctx, slot, value, prev)
	must.NoError(ctx, err)
	return r
}

func (cc *SCPControlClient) StopNomination(ctx context.Context, slot scpapi.Slot) {
	err := cc.Scp_control_stop_nomination(ctx, slot)
	must.NoError(ctx, err)
}

func (cc *SCPControlClient) UpdateQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) {
	err := cc.Scp_control_update_local_quorum_set(ctx, qset)
	must.NoError(ctx, err)
}

func (cc *SCPControlClient) Close() {
	cc.driver.Close()
}
