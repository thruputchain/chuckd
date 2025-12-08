package protoutil

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"math/rand"
	"os"
	"path"

	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func InitConfig(ctx context.Context, rr *rand.Rand, address string) *proto.ValidatorConfig {
	privInfo := MustGenerateIdentity(ctx, rr, address)
	return &proto.ValidatorConfig{Private_info: *privInfo}
}

func MustGenerateIdentity(ctx context.Context, rr *rand.Rand, address string) *proto.ValidatorPrivateInfo {
	p, err := GenerateIdentity(rr, address)
	must.NoError(ctx, err)
	return p
}

func GenerateIdentity(rr *rand.Rand, address string) (*proto.ValidatorPrivateInfo, error) {
	pubKey, secretKey, err := ed25519.GenerateKey(rr)
	if err != nil {
		return nil, err
	}
	return &proto.ValidatorPrivateInfo{
		Public_info: proto.ValidatorPublicInfo{
			Public_key: proto.ValidatorPublicKey(pubKey),
			Address:    address,
		},
		Secret_key: proto.ValidatorSecretKey(secretKey),
	}, nil
}

func GenerateConfigs(ctx context.Context, rr *rand.Rand, n int) []*proto.ValidatorConfig {
	cfg := make([]*proto.ValidatorConfig, n)
	vs := make([]proto.ValidatorPublicInfo, n)
	for i := 0; i < n; i++ {
		cfg[i] = InitConfig(ctx, rr, fmt.Sprintf("XXX.XXX.XXX.%d:9111", i))
		vs[i] = cfg[i].Private_info.Public_info
	}
	for i := range cfg {
		cfg[i].Known_validators = vs
	}
	return cfg
}

func WriteConfigsToDir(ctx context.Context, cfgs []*proto.Config, dir string) {
	err := os.MkdirAll(dir, 0755)
	must.NoError(ctx, err)
	for i, cfg := range cfgs {
		jsbuf, err := libxdr.XdrToJson(cfg)
		must.NoError(ctx, err)
		err = os.WriteFile(path.Join(dir, fmt.Sprintf("config-%d.json", i)), jsbuf, 0644)
		must.NoError(ctx, err)
	}
}
