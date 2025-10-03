package testcatchup

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestCatchupStaged struct {
	time   libtime.Time
	Signer proto.ValidatorPrivateInfo
	m      xdrutil.XdrMarshaler
	lk     sync.Mutex
	result [][]proto.CatchupBlock
}

func NewTestCatchup(time libtime.Time, signer proto.ValidatorPrivateInfo) *TestCatchupStaged {
	return &TestCatchupStaged{time: time, Signer: signer, m: xdrutil.NewXdrMarshaler()}
}

func (x *TestCatchupStaged) Catchup_null(context.Context) error {
	return nil
}

func (x *TestCatchupStaged) StageCatchup(blocks ...proto.CatchupBlock) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.result = append(x.result, blocks)
}

func (x *TestCatchupStaged) Catchup_solicit(ctx context.Context, req proto.SignedCatchupBlocksRequest) (proto.SignedCatchupBlocksResult, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	must.Assertf(ctx, len(x.result) > 0, "no staged catchup blocks")
	result := proto.CatchupBlocksResult{
		Timestamp: uint64(x.time.Now().UnixNano()),
		Blocks:    x.result[0],
	}
	r := proto.SignedCatchupBlocksResult{
		Signer:    x.Signer.Public_info,
		Signature: protoutil.SignXdr(x.m, x.Signer.Secret_key, &result),
		Result:    result,
	}
	x.result = x.result[1:]
	return r, nil
}

func (db *TestCatchupStaged) Close() {
}
