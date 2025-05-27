package loadgen

import (
	"context"

	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/system"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

type LoadGen struct {
	inside    libnet.Network
	Signer    protoutil.Signatory
	Coord     *coord.CoordinatorService
	insideSrv *system.Server
}

func NewLoadGenBench(ctx context.Context) *LoadGen {
	return &LoadGen{}
}
