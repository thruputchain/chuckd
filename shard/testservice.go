package shard

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type TestShardService struct{}

func NewTestShardService(ctx context.Context) *TestShardService {
	return &TestShardService{}
}

func (s *TestShardService) Shard_null(ctx context.Context) error {
	return nil
}

func (s *TestShardService) Shard_restore(ctx context.Context, blockHeight proto.BlockHeight) (proto.ShardResult, error) {
	return proto.ShardOK, nil
}
