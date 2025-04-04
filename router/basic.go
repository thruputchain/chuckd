package router

import (
	"context"

	"github.com/groundhogchain/chuckd/discovery"
)

type basicRouter struct {
	discovery *discovery.DiscoveryService
}

func NewBasicRouter(ctx context.Context, discovery *discovery.DiscoveryService) Router {
	return &basicRouter{
		discovery: discovery,
	}
}

func (x *basicRouter) BuildTree(ValidatorID, ChannelID) (MulticastTreeNodes, []ValidatorPublicInfo, MulticastTreeHash) {
	panic("not implemented")
}
