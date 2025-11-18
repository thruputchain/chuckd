package debug

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libnet"
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
	net libnet.AddressDialer,
	addr []string,
) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ech := make(chan error, len(addr))
	for _, addr := range addr {
		go func(addr string) {
			c, err := DialDebugClient(ctx, net, addr)
			if err != nil {
				ech <- err
				return
			}
			defer c.Close()
			ech <- c.Debug_ready(ctx)
		}(addr)
	}

	for i := 0; i < len(addr); i++ {
		err := <-ech
		if err != nil {
			cancel()
			return err
		}
	}

	return nil
}
