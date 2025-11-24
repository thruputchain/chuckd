package repairkit

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type RepairAgent interface {
	RepairMissingTxs(
		ctx context.Context,
		from proto.ValidatorPublicInfo,
		missing proto.MissingTransactions,
	) ([]proto.SignedTransaction, error)

	RepairMissingCV(
		ctx context.Context,
		from proto.ValidatorPublicInfo,
		missing proto.ConsensusValueHash,
		height proto.BlockHeight,
	) (*proto.ConsensusValue, error)
}
