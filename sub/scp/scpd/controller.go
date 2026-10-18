package scpd

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/thruputchain/chuckd/sub/scp/scpkit"
	"github.com/thruputchain/chuckd/system/systemfixed"
	"github.com/thruputchain/chuckd/util/rpcutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
	"github.com/thruputchain/lib-go/librpc"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
	"github.com/xdrpp/goxdr/rpc"
)

type SCPController struct {
	proto.SCP_Control1_Client
	driver *rpc.Driver
}

func DialSCPController(ctx context.Context, dialer libnet.Dialer, address string) (*SCPController, error) {
	sock, err := dialer.Dial(ctx, address)
	if err != nil {
		return nil, err
	}
	return NewSCPController(ctx, sock), nil
}

func NewSCPController(ctx context.Context, sock net.Conn) *SCPController {
	driver := librpc.NewDriverSerial(ctx, sock, systemfixed.KCoordinatorScpdRpcMaxBytes, rpcutil.BaseDriverRegime)
	go driver.Go()

	return &SCPController{
		SCP_Control1_Client: proto.SCP_Control1_Client{Send: driver, Ctx: ctx},
		driver:              driver,
	}
}

func (cc *SCPController) SCPReceivedEnvelope(ctx context.Context, env proto.SCPEnvelope) {
	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	err := cc.Scp_control_received_envelope(context.WithoutCancel(ctx), env)
	must.NoError(ctx, err)
}

func (cc *SCPController) SCPSetStateFromEnvelope(ctx context.Context, slot scpkit.Slot, env proto.SCPEnvelope) {
	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	err := cc.Scp_control_set_state_from_envelope(context.WithoutCancel(ctx), slot, env)
	must.NoError(ctx, err)
}

func (cc *SCPController) SCPNominate(ctx context.Context, slot scpkit.Slot, value scpkit.Value, prev scpkit.Value) bool {
	t0 := time.Now()
	defer func() {
		liblog.Debugf(ctx, "scpd_nominate_delay=%v | slot=%d", time.Since(t0), slot)
	}()

	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	r, err := cc.Scp_control_nominate(context.WithoutCancel(ctx), slot, value, prev)
	if err != nil {
		// the context can be cancelled if an externalization occurs during the call to Scp_control_nominate.
		// in this case, we return false (meaning nomination rejected).
		must.Assertf(ctx, errors.Is(err, context.Canceled), "scp control error (%v)", err)
		return false
	}
	return r
}

func (cc *SCPController) SCPStopNomination(ctx context.Context, slot scpkit.Slot) {
	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	err := cc.Scp_control_stop_nomination(context.WithoutCancel(ctx), slot)
	must.NoError(ctx, err)
}

func (cc *SCPController) SCPUpdateQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) {
	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	err := cc.Scp_control_update_local_quorum_set(context.WithoutCancel(ctx), qset)
	must.NoError(ctx, err)
}

func (cc *SCPController) SCPSetKnownQuorumSet(ctx context.Context, qset proto.SCPQuorumSet) proto.SCPOptQuorumSet {
	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	oqset, err := cc.Scp_control_set_qset(context.WithoutCancel(ctx), qset)
	must.NoError(ctx, err)
	return oqset
}

func (cc *SCPController) SCPGetKnownQuorumSet(ctx context.Context, qhash proto.SCPHash) proto.SCPOptQuorumSet {
	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	oqset, err := cc.Scp_control_get_qset(context.WithoutCancel(ctx), qhash)
	must.NoError(ctx, err)
	return oqset
}

func (cc *SCPController) SCPRemoveKnownQuorumSet(ctx context.Context, qhash proto.SCPHash) proto.SCPOptQuorumSet {
	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	oqset, err := cc.Scp_control_remove_qset(context.WithoutCancel(ctx), qhash)
	must.NoError(ctx, err)
	return oqset
}

func (cc *SCPController) SCPSetLowestNonExternalized(ctx context.Context, slot scpkit.Slot) {
	// dont cancel calls to scp; it may break the connection; they always succeed if scpd is healthy
	err := cc.Scp_control_set_lowest_non_externalized_slot(context.WithoutCancel(ctx), slot)
	must.NoError(ctx, err)
}

func (cc *SCPController) Close() {
	cc.driver.Close()
}
