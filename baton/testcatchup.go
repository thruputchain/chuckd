package baton

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestCatchup struct {
	Signer proto.ValidatorPrivateInfo
	lk     sync.Mutex
	next   *proto.CatchupBlocksResult
}

func NewTestCatchup(signer proto.ValidatorPrivateInfo) *TestCatchup {
	return &TestCatchup{Signer: signer}
}

func (x *TestCatchup) Catchup_null(context.Context) error {
	return nil
}

func (x *TestCatchup) Control_SetNextCatchupResult(result proto.CatchupBlocksResult) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.next = &result
}

func (x *TestCatchup) Catchup_blocks(ctx context.Context, req proto.SignedCatchupBlocksRequest) (proto.SignedCatchupBlocksResult, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	r := proto.SignedCatchupBlocksResult{
		Signer:    x.Signer.Public_info,
		Signature: protoutil.SignXdr(x.Signer.Secret_key, x.next),
		Result:    *x.next,
	}
	x.next = nil
	return r, nil
}

func (db *TestCatchup) Close() error {
	return nil
}
