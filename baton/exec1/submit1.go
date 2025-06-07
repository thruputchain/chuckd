package exec1

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	proto "github.com/groundhogchain/protocol-go"
)

type SubmitPkg[Tx any, Result any, Pkg any] interface {
	Buffer([]Tx) <-chan Pkg
	Submit(
		ctx context.Context,
		ctxCancel context.CancelFunc, // cancel workers context to stop the other feeders if any of them fail
		blockID proto.BlockID,
		exec batonapi.ExecCoordConn,
		payCh <-chan Pkg,
	) ([]DoneTx[Tx, Result], error)
}
