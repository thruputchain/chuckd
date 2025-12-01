package hoglock

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

type Client struct {
	Net          libnet.Network
	LockdAddress string
}

func (c *Client) Signal(ctx context.Context, xid proto.HogXID, ns proto.HogNamespaces) error {
	conn, err := DialHogLockClient(ctx, c.Net, c.LockdAddress)
	must.NoError(ctx, err)

	return conn.Hoglock_signal(ctx, xid, ns)
}

func (c *Client) Wait(ctx context.Context, xid proto.HogXID, prefix proto.HogNamespaces) error {
	conn, err := DialHogLockClient(ctx, c.Net, c.LockdAddress)
	must.NoError(ctx, err)

	return conn.Hoglock_wait(ctx, xid, prefix)
}

func (c *Client) AcquireLocks(
	ctx context.Context,
	xid proto.HogXID,
	fixed proto.HogHosts,
	variable int,
) (assigned proto.HogHosts, lease *Lease) {
	conn, err := DialHogLockClient(ctx, c.Net, c.LockdAddress)
	must.NoError(ctx, err)

	assigned, err = conn.Hoglock_lock(ctx, xid, proto.HogAsk{Fixed: fixed, Variable: int64(variable)})
	must.NoError(ctx, err)

	return assigned, makeLease(ctx, conn, xid)
}

type Lease struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   *HogLockClient
	xid    proto.HogXID
}

func makeLease(ctx context.Context, conn *HogLockClient, xid proto.HogXID) *Lease {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	x := &Lease{ctx: ctx, cancel: cancel, conn: conn, xid: xid}
	go x.renew(ctx)
	return x
}

func (x *Lease) renew(ctx context.Context) {
	for {
		time.Sleep(LeaseRenewInterval)
		select {
		case <-ctx.Done():
			x.conn.Close()
			return
		default:
		}
		err := x.conn.Hoglock_renew(ctx, x.xid)
		must.NoError(ctx, err)
	}
}

func (x *Lease) Unlock() {
	x.cancel()
}
