package repairagent

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/sub/repair/repairkit"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/librand"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

// RepairMissingTxs ensures that the returned transactions match the desired hashes.
func (x *RepairAgent) RepairMissingTxs(
	ctx context.Context,
	proposer proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) ([]proto.SignedTransaction, error) {

	backoff := netutil.NewBackoff(x.backoffCfg, librand.NewRand())

attempting:
	for {
		txs, err := x.attemptRepairMissingTxs(ctx, proposer, missing)
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
	proposer proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) (txs []proto.SignedTransaction, err error) {

	ctx = liblog.Enterf(ctx, "RepairMissingTxs", "proposer=%s; height=%d; cv_hash=%s",
		protoutil.VPublicShortString(proposer), missing.Height, libxdr.HashString(missing.Cv_hash))
	defer func() {
		liblog.Leavef(ctx, "proposer=%s; height=%d; cv_hash=%s; error=%v; found=%d",
			protoutil.VPublicShortString(proposer), missing.Height, libxdr.HashString(missing.Cv_hash),
			err, len(txs))
	}()

	c, err := repairkit.DialRepairClient(ctx, x.net, proposer.Address)
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
		liblog.Errorf(ctx, "unexpected repaired transaction count")
		return nil, fmt.Errorf("unexpected repaired transaction count")
	}
	for i, tx := range reply.Transactions {
		if protoutil.HashSignedTx(&tx) != missing.Transaction_hashes[i] {
			liblog.Errorf(ctx, "repaired transaction hash mismatch")
			return nil, fmt.Errorf("repaired transaction hash mismatch")
		}
	}

	return reply.Transactions, nil
}
