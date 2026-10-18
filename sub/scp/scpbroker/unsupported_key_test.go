package scpbroker

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/thruputchain/chuckd/sub/scp/scpkit"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type fixedRepairer struct{ envelopes proto.SCPEnvelopes }

func (r fixedRepairer) RepairMissingSCP(context.Context, scpkit.Slot) (proto.SCPEnvelopes, error) {
	return r.envelopes, nil
}

func TestBrokerRejectsUnsupportedPublicKey(t *testing.T) {
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(oldLevel) })
	env := proto.SCPEnvelope{Statement: proto.SCPStatement{
		Slot_index: 1,
		Node_id:    proto.PublicKey{Scheme: proto.CryptographicScheme(123)},
	}}
	for _, backfill := range []bool{false, true} {
		var logs bytes.Buffer
		ctx := zerolog.New(&logs).WithContext(context.Background())
		// Nil storage, router, and controller must never be reached.
		broker := NewSCPBroker(ctx, nil, nil, fixedRepairer{proto.SCPEnvelopes{env}})
		broker.SetSCPController(nil)
		if backfill {
			broker.fulfillBackfeed(ctx, 1, time.Second)
		} else {
			broker.SCPFeedEnvelope(ctx, env)
		}
		broker.Stop()
		if !strings.Contains(logs.String(), `"level":"info"`) || !strings.Contains(logs.String(), "unsupported public key scheme") {
			t.Fatalf("backfill=%v: expected Info rejection log, got %s", backfill, logs.String())
		}
	}
}
