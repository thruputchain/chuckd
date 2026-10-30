package protoutil

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/thruputchain/chuckd/util/configtext"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func InitConfig(ctx context.Context, rr io.Reader, address string) *proto.ValidatorConfig {
	privInfo := MustGenerateIdentity(ctx, rr, address)
	return &proto.ValidatorConfig{Private_info: *privInfo}
}

func MustGenerateIdentity(ctx context.Context, rr io.Reader, address string) *proto.ValidatorPrivateInfo {
	p, err := GenerateIdentity(rr, address)
	must.NoError(ctx, err)
	return p
}

func GenerateIdentity(rr io.Reader, address string) (*proto.ValidatorPrivateInfo, error) {
	pubKey, secretKey, err := ed25519.GenerateKey(rr)
	if err != nil {
		return nil, err
	}
	publicKey := proto.ValidatorPublicKey{Scheme: proto.Ed25519}
	*publicKey.Ed25519() = proto.Ed25519PublicKey(pubKey)
	privateKey := proto.ValidatorSecretKey{Scheme: proto.Ed25519}
	*privateKey.Ed25519() = proto.Ed25519SecretKey(secretKey)
	return &proto.ValidatorPrivateInfo{
		Public_info: proto.ValidatorPublicInfo{
			Public_key: publicKey,
			Address:    address,
		},
		Secret_key: privateKey,
	}, nil
}

func GenerateConfigs(ctx context.Context, rr io.Reader, n int) []*proto.ValidatorConfig {
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

// WriteConfigsToDir writes config-<i>.json for each config. The files hold
// secret keys, so they and a directory this creates are readable by this
// user only, and an existing file is refused rather than overwritten.
func WriteConfigsToDir(ctx context.Context, cfgs []*proto.Config, dir string) {
	err := os.MkdirAll(dir, 0o700)
	must.NoError(ctx, err)
	for i, cfg := range cfgs {
		jsbuf, err := configtext.Marshal(cfg)
		must.NoError(ctx, err)
		err = WriteNewConfigFile(path.Join(dir, fmt.Sprintf("config-%d.json", i)), jsbuf)
		must.NoError(ctx, err)
	}
}

// WriteNewConfigFile creates path with mode 0600 and writes text to it. It
// refuses to replace an existing file: writing into one would keep whatever
// permissions it has, exposing the secret key a config holds.
func WriteNewConfigFile(path string, text []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("config already exists: %s", path)
		}
		return err
	}
	if _, err := f.Write(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
