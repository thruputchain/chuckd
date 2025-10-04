package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) loadTxs(
	ctx context.Context,
	proposer proto.ValidatorPublicInfo,
	blockHash proto.BlockHash,
	hashes []proto.SignedTransactionHash,
) ([]*proto.SignedTransaction, error) {

	txs := b.handler.PartialLookupTxs(ctx, hashes)
	missingTxs := []proto.SignedTransactionHash{}
	for i, tx := range txs {
		if tx == nil {
			missingTxs = append(missingTxs, hashes[i])
		}
	}
	if len(missingTxs) == 0 {
		return txs, nil
	}

	missing := proto.MissingTransactions{
		Transaction_hashes: missingTxs,
		Block_hash:         blockHash,
	}

	rtxs, err := b.handler.RepairMissingTxs(ctx, proposer, missing)
	if err != nil {
		// this can only ever be context cancelled
		return nil, err
	}

	// places the repaired txs in their slots
	for i, tx := range txs {
		if tx == nil {
			txs[i] = &rtxs[0]
			rtxs = rtxs[1:]
		}
	}

	return protoutil.Unpack(rtxs), nil
}
