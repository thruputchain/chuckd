package dogwood

import (
	"context"

	proto "github.com/thruputchain/thruput-protocol-go"
)

func (x *DogwoodService) Dogwood_null(ctx context.Context) error {
	return nil
}

func (x *DogwoodService) Dogwood_deliver(ctx context.Context, env proto.DogwoodDeliveryEnvelope) error {
	x.streamReceiver.Deliver(ctx, env)
	return nil
}
