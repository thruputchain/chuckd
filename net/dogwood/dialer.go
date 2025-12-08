package dogwood

import (
	"context"

	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/lib-go/libnet"
	proto "github.com/groundhogchain/protocol-go"
)

type DogwoodDialer interface {
	Dial(context.Context, proto.ValidatorID) (DogwoodClientCloser, error)
	Start(context.Context)
}

type dogwoodClientDialer struct {
	net    libnet.Network
	router router.Router
}

func (x dogwoodClientDialer) Start(context.Context) {}

func (x dogwoodClientDialer) Dial(ctx context.Context, vid proto.ValidatorID) (DogwoodClientCloser, error) {

	addr, err := x.router.ResolveAddress(vid)
	if err != nil {
		return nil, err
	}
	c, err := DialDogwoodClient(ctx, x.net, addr)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func NewDogwoodClientDialer(net libnet.Network, router router.Router) DogwoodDialer {
	return dogwoodClientDialer{net: net, router: router}
}
