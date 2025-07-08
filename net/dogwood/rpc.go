package dogwood

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (x *DogwoodService) Dogwood_null(ctx context.Context) error {
	return nil
}

func (x *DogwoodService) Dogwood_deliver(ctx context.Context, env proto.DogwoodDeliveryEnvelope) error {
	panic("XXX")
}
