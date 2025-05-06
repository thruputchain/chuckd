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

type liveHandler struct {
	coordinator *coord.CoordinatorService
	spectra     *spectra.SpectraTransport
	mempool     *mempool.ProposedMempool
	scp         api.SCPController
	wallet      *wallet.WalletGateway
}

// XXX: hookup scp to the baton as handler, after initialization
func NewLiveHandler(
	ctx context.Context,
	coord *coord.CoordinatorService,
	spectra *spectra.SpectraTransport,
	mp *mempool.ProposedMempool,
	scp api.SCPController,
	w *wallet.WalletGateway,
) *liveHandler {
	x := &liveHandler{
		coordinator: coord,
		spectra:     spectra,
		mempool:     mp,
		scp:         scp,
		wallet:      w,
	}
	return x
}
