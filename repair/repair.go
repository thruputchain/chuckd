package repair

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairAgent struct {
	cfg       proto.RepairConfig
	net       libnet.AddressDialer
	signatory protoutil.Signatory
	m         xdrutil.XdrMarshaler
}

func NewRepairAgent(
	cfg proto.RepairConfig,
	net libnet.AddressDialer,
	signatory protoutil.Signatory,
) *RepairAgent {
	return &RepairAgent{
		cfg:       cfg,
		net:       net,
		signatory: signatory,
		m:         xdrutil.NewXdrMarshaler(),
	}
}

func (x *RepairAgent) RepairMissingTxs(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) ([]proto.SignedTransaction, error) {

	cfg := netutil.BackoffConfig{
		BaseDelay:  time.Duration(x.cfg.Base_delay_msec) * time.Millisecond,
		MaxDelay:   time.Duration(x.cfg.Max_delay_msec) * time.Millisecond,
		Multiplier: float64(x.cfg.Multiplier_nom) / float64(x.cfg.Multiplier_denom),
		MaxRetries: 0,
		JitterType: netutil.EqualJitter,
	}
	backoff := netutil.NewBackoff(cfg, librand.New())

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

	c, err := DialRepairClient(ctx, x.net, from.Address)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	env := proto.MissingTransactionsEnvelope{
		Signer:               x.signatory.Signer().Public_key,
		Signature:            x.signatory.SignXdr(x.m, &missing),
		Missing_transactions: missing,
	}
	reply, err := c.Repair_missing_txs(ctx, env)
	if err != nil {
		return nil, err
	}

	// verify returned txs match hashes
	if len(reply.Transactions) != len(missing.Transaction_hashes) {
		return nil, fmt.Errorf("unexpected repaired transaction count")
	}
	for i, tx := range reply.Transactions {
		if protoutil.HashSignedTx(x.m, &tx) != missing.Transaction_hashes[i] {
			return nil, fmt.Errorf("repaired transaction hash mismatch")
		}
	}

	return reply.Transactions, nil
}
