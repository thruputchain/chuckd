package aws

import (
	"context"
	"math/rand"
	"testing"
)

func TestAWSTestClusterConfigs(t *testing.T) {
	ctx := context.Background()
	cfgs := AWSTestClusterConfigs(
		ctx,
		rand.New(rand.NewSource(0)),
		"us-east-1",
		[]string{"us-west-1", "ap-northeast-1", "eu-central-1"},
		1000,
	)
	WriteFiles(ctx, "/Users/petar/groundhog/chuckd/deploy/aws/terraform/configs", cfgs)
}
