package dogwood

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/net/router"
)

type DogwoodService struct {
	router         router.DogwoodRouter
	dialer         DogwoodDialer
	nav            *Navigator
	caller         Caller
	streamReceiver *StreamReceiver
	streamSender   *StreamSender
	xReceiver      *TransmissionReceiver
	xSender        *TransmissionSender
	mplex          *Multiplex
}

func NewDogwoodService(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	net libnet.Network,
	router router.DogwoodRouter,
) *DogwoodService {

	x := &DogwoodService{}
	x.router = router
	x.dialer = NewDogwoodClientDialer(net, router)
	// x.nav = NewNavigator(ctx, time, randXXX, x.router, X, X, X)

	return x
}

func (x *DogwoodService) Start(ctx context.Context) {
	// XXX
}
