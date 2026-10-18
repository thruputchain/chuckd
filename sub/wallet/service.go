package wallet

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/thruputchain/chuckd/sub/kit"
	"github.com/thruputchain/chuckd/util/protoutil"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type WalletService struct {
	submitter kit.SignedTxSubmitter
	inflight  sync.WaitGroup
	stopped   atomic.Bool
}

func NewWalletService(
	ctx context.Context,
	submitter kit.SignedTxSubmitter,
) *WalletService {
	return &WalletService{
		submitter: submitter,
	}
}

func (svc *WalletService) Stop() {
	svc.stopped.Store(true)
	svc.inflight.Wait()
}

func (svc *WalletService) IsStopped() bool {
	return svc.stopped.Load()
}

func (svc *WalletService) Wallet_null(context.Context) error {
	return nil
}

func (svc *WalletService) Wallet_submit(ctx context.Context, submit proto.WalletSubmit) (proto.WalletAck, error) {

	svc.inflight.Add(1)
	defer svc.inflight.Done()

	if svc.IsStopped() {
		return proto.WalletAck{}, errors.New("wallet service is stopped")
	}

	svc.submitter.SubmitTxs(ctx, protoutil.Unpack(submit.Signed_transactions)...)

	acks := make([]proto.WalletTransactionAck, len(submit.Signed_transactions))
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
