// Package baton implements the main orchestration logic between SCP, database, executors, and wallet.
package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/broadcast"
	"github.com/groundhogchain/chuckd/coord"
	"github.com/groundhogchain/chuckd/mempool"
	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/wallet"
)

type Baton struct {
	coordinator *coord.CoordinatorService
	broadcast   broadcast.BulkBroadcast
	mempool     *mempool.Mempool
	scp         api.SCPController
	wallet      *wallet.WalletGateway
}

func NewBaton(
	ctx context.Context,
	coord *coord.CoordinatorService,
	bcast broadcast.BulkBroadcast,
	mp *mempool.Mempool,
	scp api.SCPController,
	w *wallet.WalletGateway,
) *Baton {
	x := &Baton{
		coordinator: coord,
		broadcast:   bcast,
		mempool:     mp,
		scp:         scp,
		wallet:      w,
	}
	go x.start(ctx)
	return x
}

func (x *Baton) start(ctx context.Context) {
	panic("XXX")
}
