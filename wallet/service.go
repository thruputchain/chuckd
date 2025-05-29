package wallet

import (
	"context"

	"github.com/groundhogchain/chuckd/mempool"
	proto "github.com/groundhogchain/protocol-go"
)

type WalletService struct {
	mp *mempool.SubmittedMempool
}

func NewWalletService(
	ctx context.Context,
	mp *mempool.SubmittedMempool,
) *WalletService {
	return &WalletService{
		mp: mp,
	}
}

func (svc *WalletService) Wallet_null(context.Context) error {
	return nil
}

func (svc *WalletService) Wallet_submit(ctx context.Context, submit proto.WalletSubmit) (proto.WalletAck, error) {

	svc.mp.AddTxs(ctx, submit.Signed_transactions)

	acks := []proto.WalletTransactionAck{}
	for i := range submit.Signed_transactions {
		// XXX: check signatures
		var ack proto.WalletTransactionAck
		ack.Ack_type = proto.WalletTransactionSubmitted
		// XXX: set ack.Submitted_transaction_id()
		acks[i] = ack
	}

	return proto.WalletAck{
		Transaction_acks: acks,
	}, nil
}
