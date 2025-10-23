package repairagent

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/repair/repairkit"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// RepairMissingTxs ensures that the returned transactions match the desired hashes.
func (x *RepairAgent) RepairMissingTxs(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) ([]proto.SignedTransaction, error) {

	backoff := netutil.NewBackoff(x.backoffCfg, librand.New())

attempting:
	for {
		txs, err := x.attemptRepairMissingTxs(ctx, from, missing)
		if err != nil {
			delay, canRetry := backoff.NextDelay()
			must.Assert(ctx, canRetry)
			tmo := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-tmo.C:
				continue attempting
			}
		}
		return txs, nil
	}
}

func (x *RepairAgent) attemptRepairMissingTxs(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) ([]proto.SignedTransaction, error) {

	c, err := repairkit.DialRepairClient(ctx, x.net, from.Address)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	env := proto.MissingTransactionsEnvelope{
		Signer:               x.signatory.Signer().Public_key,
		Signature:            x.signatory.SignXdr(x.txM, &missing),
		Missing_transactions: missing,
	}
	reply, err := c.Repair_missing_txs(ctx, env)
	if err != nil {
		return nil, err
	}

	// verify returned txs match hashes
	if len(reply.Transactions) != len(missing.Transaction_hashes) {
		liblog.Errorf("unexpected repaired transaction count")
		return nil, fmt.Errorf("unexpected repaired transaction count")
	}
	for i, tx := range reply.Transactions {
		if protoutil.HashSignedTx(x.txM, &tx) != missing.Transaction_hashes[i] {
			liblog.Errorf("repaired transaction hash mismatch")
			return nil, fmt.Errorf("repaired transaction hash mismatch")
		}
	}

	return reply.Transactions, nil
}
