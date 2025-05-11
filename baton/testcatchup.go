package baton

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type TestCatchup struct{}

func NewTestCatchup() *TestCatchup {
	return &TestCatchup{}
}

func (x *TestCatchup) Catchup_null(context.Context) error {
	XXX
}

func (x *TestCatchup) Catchup_blocks(context.Context, proto.SignedCatchupBlocksRequest) (proto.SignedCatchupBlocksResult, error) {
	XXX
}

func (db *TestCatchup) Close() error {
	return nil
}
