package repairagent

import (
	"context"
	"fmt"
	"time"

	"github.com/thruputchain/chuckd/sub/repair/repairkit"
	"github.com/thruputchain/chuckd/util/netutil"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/librand"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// RepairMissingTxs ensures that the returned transactions match the desired hashes.
func (x *RepairAgent) RepairMissingTxs(
	ctx context.Context,
	proposer proto.ValidatorID,
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
	proposer proto.ValidatorID,
	missing proto.MissingTransactions,
) (txs []proto.SignedTransaction, err error) {

	ctx = liblog.Enterf(ctx, "RepairMissingTxs", "proposer=%s; height=%d; value=%s",
		protoutil.VIDString(proposer), protoutil.ValueHeight(&missing.Consensus_value), protoutil.ValueDisplay(&missing.Consensus_value))
	defer func() {
		liblog.Leavef(ctx, "proposer=%s; height=%d; value=%s; error=%v; found=%d",
			protoutil.VIDString(proposer), protoutil.ValueHeight(&missing.Consensus_value), protoutil.ValueDisplay(&missing.Consensus_value),
			err, len(txs))
	}()

	address, err := x.router.ResolveAddress(proposer)
	if err != nil {
		return nil, fmt.Errorf("resolving address for %s: %w", protoutil.VIDString(proposer), err)
	}

	c, err := repairkit.DialRepairClient(ctx, x.net, address)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	env := proto.MissingTransactionsEnvelope{
		Signer:               protoutil.PublicVID(x.signatory.Signer()),
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
