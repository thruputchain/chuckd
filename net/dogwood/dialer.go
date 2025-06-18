package dogwood

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libnet"
	proto "github.com/groundhogchain/protocol-go"
)

type DogwoodClientDialer interface {
	Dial(context.Context, proto.ValidatorID) (DogwoodClientCloser, error)
}

type ResolveFn func(proto.ValidatorID) (string, error)

type dogwoodClientDialer struct {
	net     libnet.Network
	resolve ResolveFn
}

func (x dogwoodClientDialer) Dial(ctx context.Context, vid proto.ValidatorID) (DogwoodClientCloser, error) {

	addr, err := x.resolve(vid)
	if err != nil {
		return nil, err
	}
	c, err := DialDogwoodClient(ctx, x.net, addr)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func NewDogwoodClientDialer(net libnet.Network, resolve ResolveFn) DogwoodClientDialer {
	return dogwoodClientDialer{net: net, resolve: resolve}
}
