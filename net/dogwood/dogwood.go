package dogwood

import "context"

type DogwoodService struct {
	dialer DogwoodDialer
	nav    *Navigator
}

func (svc *DogwoodService) Start(ctx context.Context) {
}
