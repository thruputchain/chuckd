package dogwood

import (
	"context"

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
	router router.DogwoodRouter,
) *DogwoodService {

	return &DogwoodService{
		router: router,
		// dialer: XXX,
		// nav:    XXX,
	}
}

func (x *DogwoodService) Start(ctx context.Context) {
	// XXX
}
