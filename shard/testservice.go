package shard

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type TestShardService struct{}

func NewTestShardService(ctx context.Context) *TestShardService {
	return &TestShardService{}
}

func (s *TestShardService) Shard_null() {}

func (s *TestShardService) Shard_restore(blockHeight proto.BlockHeight) proto.ShardResult {
	return proto.ShardOK
}
