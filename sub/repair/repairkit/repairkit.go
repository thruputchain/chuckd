package repairkit

import (
	"context"

	proto "github.com/thruputchain/thruput-protocol-go"
)

type RepairAgent interface {
	RepairMissingTxs(
		ctx context.Context,
		from proto.ValidatorID,
		missing proto.MissingTransactions,
	) ([]proto.SignedTransaction, error)

	RepairMissingBasis(
		ctx context.Context,
		from proto.ValidatorID,
		missing proto.ConsensusValue,
	) (*proto.ConsensusBasis, error)
}
