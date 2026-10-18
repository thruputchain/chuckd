package dogwood

import (
	"context"
	"time"

	"github.com/thruputchain/lib-go/libmem"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func (x *DogwoodService) Send(ctx context.Context, pay libmem.BorrowedBytes) error {
	return x.mplex.Send(ctx, pay)
}

func (x *DogwoodService) Receive(ctx context.Context) ([]byte, proto.ValidatorID, error) {
	return x.mplex.Receive(ctx)
}

func (x *DogwoodService) Dogwoodapi_null(context.Context) error {
	return nil
}

func (x *DogwoodService) Dogwoodapi_send(ctx context.Context, timeout proto.Milliseconds, pay proto.Bytes) error {
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*time.Duration(timeout))
	defer cancel()
	return x.Send(ctx, libmem.BorrowBytes(pay))
}

func (x *DogwoodService) Dogwoodapi_receive(ctx context.Context, timeout proto.Milliseconds) (proto.DogwoodReceived, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*time.Duration(timeout))
	defer cancel()
	pay, from, err := x.Receive(ctx)
	if err != nil {
		return proto.DogwoodReceived{}, err
	}
	return proto.DogwoodReceived{
		From:    from,
		Payload: pay,
	}, nil
}
