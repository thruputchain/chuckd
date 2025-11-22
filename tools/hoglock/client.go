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

func (c *Client) Signal(ctx context.Context, ns proto.HogNamespaces) error {
	conn, err := DialHogLockClient(ctx, c.Net, c.LockdAddress)
	must.NoError(ctx, err)

	return conn.Hoglock_signal(ctx, ns)
}

func (c *Client) Wait(ctx context.Context, prefix proto.HogNamespaces) error {
	conn, err := DialHogLockClient(ctx, c.Net, c.LockdAddress)
	must.NoError(ctx, err)

	return conn.Hoglock_wait(ctx, prefix)
}

func (c *Client) AcquireLocks(ctx context.Context, o proto.LockObjects) *Lease {
	conn, err := DialHogLockClient(ctx, c.Net, c.LockdAddress)
	must.NoError(ctx, err)

	h, err := conn.Hoglock_lock(ctx, o)
	must.NoError(ctx, err)

	return makeLease(ctx, conn, h)
}

type Lease struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   *HogLockClient
	lease  proto.LockLease
}

func makeLease(ctx context.Context, conn *HogLockClient, h proto.LockLease) *Lease {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	x := &Lease{ctx: ctx, cancel: cancel, conn: conn, lease: h}
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
		err := x.conn.Hoglock_renew(ctx, x.lease)
		must.NoError(ctx, err)
	}
}

func (x *Lease) Unlock() {
	x.cancel()
}
