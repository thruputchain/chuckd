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

	return &DogwoodService{
		router: router,
		dialer: NewDogwoodClientDialer(net, router),
		// nav:    XXX,
	}
}

func (x *DogwoodService) Start(ctx context.Context) {
	// XXX
}
