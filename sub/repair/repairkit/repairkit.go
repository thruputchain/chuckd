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

	RepairMissingBasis(
		ctx context.Context,
		from proto.ValidatorPublicInfo,
		missing proto.ConsensusValue,
	) (*proto.ConsensusBasis, error)
}
