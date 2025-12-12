package scpd

import (
	"context"
	"errors"
	"net"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SCPController struct {
	proto.SCP_Control1_Client
	driver *rpc.Driver
}

func DialSCPController(ctx context.Context, dialer libnet.AddressDialer, address string) (*SCPController, error) {
	sock, err := dialer.DialAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewSCPController(ctx, sock), nil
}

func NewSCPController(ctx context.Context, sock net.Conn) *SCPController {
	driver := librpc.NewDriverSerial(ctx, sock, netutil.MaxMessageSize)
	go driver.Go()

	return &SCPController{
		SCP_Control1_Client: proto.SCP_Control1_Client{Send: driver, Ctx: ctx},
		driver:              driver,
	}
}

func (cc *SCPController) ReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	err := cc.Scp_control_received_envelope(ctx, env)
	must.NoError(ctx, err)
}

func (cc *SCPController) Nominate(ctx context.Context, slot scpkit.Slot, value scpkit.Value, prev scpkit.Value) bool {
	r, err := cc.Scp_control_nominate(ctx, slot, value, prev)
	if err != nil {
		// the context can be cancelled if an externalization occurs during the call to Scp_control_nominate.
		// in this case, we return false (meaning nomination rejected).
		must.Assertf(ctx, errors.Is(err, context.Canceled), "scp control error (%v)", err)
		return false
	}
	return r
}

func (cc *SCPController) StopNomination(ctx context.Context, slot scpkit.Slot) {
	err := cc.Scp_control_stop_nomination(ctx, slot)
	must.NoError(ctx, err)
}

func (cc *SCPController) UpdateQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) {
	err := cc.Scp_control_update_local_quorum_set(ctx, qset)
	must.NoError(ctx, err)
}

func (cc *SCPController) Close() {
	cc.driver.Close()
}
