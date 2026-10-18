package testcatchup

import (
	"context"
	"sync"

	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/libtime"
	"github.com/thruputchain/lib-go/libxdr"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type TestCatchupStaged struct {
	time   libtime.Time
	Signer proto.ValidatorPrivateInfo
	m      libxdr.XdrMarshaler
	lk     sync.Mutex
	result [][]proto.CatchupBlock
}

func NewTestCatchup(time libtime.Time, signer proto.ValidatorPrivateInfo) *TestCatchupStaged {
	return &TestCatchupStaged{time: time, Signer: signer, m: libxdr.NewXdrMarshaler()}
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
		Timestamp: protoutil.TimepointFromTime(x.time.Now()),
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
