package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (h *TestRig) MakeSignedTx(ctx context.Context) *proto.SignedTransaction {
	h.rlk.Lock()
	defer h.rlk.Unlock()

	var id proto.OriginID
	h.rand.Read(id[:])

	tx := proto.Transaction{
		Id:      id,
		Created: h.TestTime.Now().UnixNano(),
	}
	return &proto.SignedTransaction{
		Signature: protoutil.SignXdr(h.user, &tx),
		Tx:        tx,
	}
}
