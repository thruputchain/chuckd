package repair

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type RepairAgent struct {
	net       libnet.AddressDialer
	signatory protoutil.Signatory
	m         xdrutil.XdrMarshaler
}

func NewRepairAgent(
	net libnet.AddressDialer,
	signatory protoutil.Signatory,
) *RepairAgent {
	return &RepairAgent{
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

attempting:
	for {
		txs, err := x.attemptRepairMissingTxs(ctx, from, missing)
		if err != nil {
			panic("XXX")
			XXX := time.Second
			tmo := time.NewTimer(XXX) //backoff
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

	env := proto.MissingTransactionsEnvelope{
		Signer:               x.signatory.Signer().Public_key,
		Signature:            x.signatory.SignXdr(x.m, &missing),
		Missing_transactions: missing,
	}
	reply, err := c.Repair_missing_txs(ctx, env)
	if err != nil {
		return nil, err
	}
	_ = reply

	// XXX: verify returned txs match hashes

	panic("XXX")
}
