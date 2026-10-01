package common

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func genTestConfigs(t *testing.T) []proto.Config {
	t.Helper()
	base := MakeBaseConfig(time.Second, 2, 5)
	cfgs, _ := GenQuorumConfigs(
		context.Background(),
		rand.New(rand.NewSource(1)),
		base,
		[]string{"10.0.0.1:50000", "10.0.0.2:50000"},
	)
	if len(cfgs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(cfgs))
	}
	return cfgs
}

// A generated config has to survive the trip through JSON that every
// checked-in config file takes: coordinators load theirs with JsonToXdr at
// startup. A protocol change that reshapes a config field (a plain value
// becoming a union, say) breaks that decode, and the only symptom is a
// coordinator that panics before it registers.
func TestGenQuorumConfigsSurvivesJsonRoundTrip(t *testing.T) {
	for i, cfg := range genTestConfigs(t) {
		encoded, err := libxdr.XdrToJson(&cfg)
		if err != nil {
			t.Fatalf("config %d: encoding to JSON: %v", i, err)
		}

		var decoded proto.Config
		if err := libxdr.JsonToXdr(&decoded, encoded); err != nil {
			t.Fatalf("config %d: decoding %s: %v", i, encoded, err)
		}

		reencoded, err := libxdr.XdrToJson(&decoded)
		if err != nil {
			t.Fatalf("config %d: re-encoding to JSON: %v", i, err)
		}
		if string(reencoded) != string(encoded) {
			t.Fatalf("config %d: round trip changed the config\n got: %s\nwant: %s",
				i, reencoded, encoded)
		}
	}
}

// The same seed and addresses have to keep producing the same identities, or
// regenerating a checked-in config silently rotates its keys.
func TestGenQuorumConfigsIsDeterministic(t *testing.T) {
	first, second := genTestConfigs(t), genTestConfigs(t)
	for i := range first {
		a, err := libxdr.XdrToJson(&first[i])
		if err != nil {
			t.Fatalf("config %d: encoding first run: %v", i, err)
		}
		b, err := libxdr.XdrToJson(&second[i])
		if err != nil {
			t.Fatalf("config %d: encoding second run: %v", i, err)
		}
		if string(a) != string(b) {
			t.Fatalf("config %d: same seed produced different configs\nfirst:  %s\nsecond: %s",
				i, a, b)
		}
	}
}
