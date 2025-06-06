package exec

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	proto "github.com/groundhogchain/protocol-go"
)

type SubmitPayload[Tx any, Result any, Payload any] interface {
	Buffer([]Tx) <-chan Payload
	Submit(
		ctx context.Context,
		ctxCancel context.CancelFunc, // cancel workers context to stop the other feeders if any of them fail
		blockID proto.BlockID,
		exec batonapi.ExecCoordConn,
		payCh <-chan Payload,
	) ([]DoneTx[Tx, Result], error)
}
