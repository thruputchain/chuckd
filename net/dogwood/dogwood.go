package dogwood

import (
	"context"

	"github.com/groundhogchain/chuckd/net/router"
)

type DogwoodService struct {
	router router.DogwoodRouter
	dialer DogwoodDialer
	nav    *Navigator
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

func (svc *DogwoodService) Start(ctx context.Context) {
	// XXX
}
