package catchup

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	proto "github.com/thruputchain/thruput-protocol-go"
)

func TestCatchupRejectsUnsupportedPublicKey(t *testing.T) {
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(oldLevel) })
	var logs bytes.Buffer
	ctx := zerolog.New(&logs).WithContext(context.Background())
	// Leave the pool and marshaler unset: neither should be reached.
	svc := &CatchupService{}
	_, err := svc.Catchup_solicit(ctx, proto.SignedCatchupBlocksRequest{
		Signer: proto.ValidatorPublicInfo{Public_key: proto.PublicKey{Scheme: proto.CryptographicScheme(123)}},
	})
	if err == nil {
		t.Fatal("accepted unsupported catchup signer")
	}
	if !strings.Contains(logs.String(), `"level":"info"`) || !strings.Contains(logs.String(), "unsupported public key scheme") {
		t.Fatalf("expected Info rejection log, got %s", logs.String())
	}
}
