package debug

import (
	"context"
	"time"

	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/libnet"
)

func StartAndWaitCounterpartiesReady(
	ctx context.Context,
	net libnet.Network,
	bind string,
	addr []string,
) error {

	svc := NewDebugService()
	svc.Start(context.WithoutCancel(ctx), net, bind)

	return WaitCounterpartiesReady(ctx, net, addr)
}

func WaitCounterpartiesReady(
	ctx context.Context,
	net libnet.Dialer,
	addr []string,
) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ech := make(chan error, len(addr))
	for _, addr := range addr {
		go func(ctx context.Context, addr string) {
			for {
				select {
				case <-ctx.Done():
					ech <- ctx.Err()
					return
				default:
				}

				c, err := DialDebugClient(ctx, net, addr)
				if err != nil {
					liblog.Debugf(ctx, "failed to dial debug counterparty %s: %v", addr, err)
					time.Sleep(time.Second)
					continue
				}
				defer c.Close()

				ech <- c.Debug_ready(ctx)
				return
			}
		}(ctx, addr)
	}

	for range len(addr) {
		err := <-ech
		if err != nil {
			cancel()
			return err
		}
	}

	return nil
}
