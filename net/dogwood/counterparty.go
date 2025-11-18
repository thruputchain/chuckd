package dogwood

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	proto "github.com/groundhogchain/protocol-go"
)

func WaitCounterpartiesReady(ctx context.Context, dialer libnet.AddressDialer, vs []proto.ValidatorPublicInfo) {
	var wg sync.WaitGroup
	for _, v := range vs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			WaitCounterpartyReady(ctx, dialer, v)
		}()
	}
	wg.Wait()
}

func WaitCounterpartyReady(ctx context.Context, dialer libnet.AddressDialer, v proto.ValidatorPublicInfo) error {
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if i > 0 {
				time.Sleep(time.Second)
			}
			err := pingDowgwood(ctx, dialer, v.Address)
			if err != nil {
				liblog.Errorf("pinging dogwood service at %s failed (%v)", v.Address, err)
				continue
			}
			return nil
		}
	}
}

func pingDowgwood(ctx context.Context, dialer libnet.AddressDialer, address string) error {
	c, err := DialDogwoodClient(ctx, dialer, address)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.Dogwood_null(ctx)
}
