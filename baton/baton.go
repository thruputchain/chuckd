// Package baton implements the main orchestration logic between SCP, database, executors, and wallet.
package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/spectra"
	"github.com/groundhogchain/chuckd/wallet"
)

type Baton struct {
	coordinator *coord.CoordinatorService
	spectra     *spectra.SpectraTransport
	mempool     *mempool.Mempool
	scp         api.SCPController
	wallet      *wallet.WalletGateway
}

// XXX: hookup scp to the baton as handler, after initialization
func NewBaton(
	ctx context.Context,
	coord *coord.CoordinatorService,
	spectra *spectra.SpectraTransport,
	mp *mempool.Mempool,
	scp api.SCPController,
	w *wallet.WalletGateway,
) *Baton {
	x := &Baton{
		coordinator: coord,
		spectra:     spectra,
		mempool:     mp,
		scp:         scp,
		wallet:      w,
	}
	go x.start(ctx)
	return x
}

func (x *Baton) start(ctx context.Context) {

	// wait until databases are ready

	panic("XXX")
}
