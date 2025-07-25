package dogwood

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (x *DogwoodService) Send(ctx context.Context, pay []byte) error {
	return x.mplex.Send(ctx, pay)
}

func (x *DogwoodService) Receive(ctx context.Context) ([]byte, proto.ValidatorID, error) {
	return x.mplex.Receive(ctx)
}
