package baton

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestCatchup struct {
	time   libtime.Time
	Signer proto.ValidatorPrivateInfo
	lk     sync.Mutex
	result [][]proto.CatchupBlock
}

func NewTestCatchup(time libtime.Time, signer proto.ValidatorPrivateInfo) *TestCatchup {
	return &TestCatchup{time: time, Signer: signer}
}

func (x *TestCatchup) Catchup_null(context.Context) error {
	return nil
}

func (x *TestCatchup) StageCatchup(blocks ...proto.CatchupBlock) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.result = append(x.result, blocks)
}

func (x *TestCatchup) Catchup_solicit(ctx context.Context, req proto.SignedCatchupBlocksRequest) (proto.SignedCatchupBlocksResult, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	must.Assertf(ctx, len(x.result) > 0, "no staged catchup blocks")
	result := proto.CatchupBlocksResult{
		Timestamp: x.time.Now().UnixNano(),
		Blocks:    x.result[0],
	}
	r := proto.SignedCatchupBlocksResult{
		Signer:    x.Signer.Public_info,
		Signature: protoutil.SignXdr(x.Signer.Secret_key, &result),
		Result:    result,
	}
	x.result = x.result[1:]
	return r, nil
}

func (db *TestCatchup) Close() error {
	return nil
}
