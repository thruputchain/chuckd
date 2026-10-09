package scpd

import (
	"reflect"
	"strings"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestSCPDEnv(t *testing.T) {
	hash := proto.NetworkHash{0xab, 0xcd}
	for _, tc := range []struct {
		name string
		hash *proto.NetworkHash
	}{
		{"without hash", nil},
		{"with hash", &hash},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := []string{"PATH=/bin", "HOME=/home", "OTELX=1", "MY_OTEL_FOO=1", "RUST_LOG=wrong", "OTEL_MALFORMED", "OTEL_SPAN_FILTER=scp=trace", "OTEL_EXPORTER_OTLP_ENDPOINT=http://collector", "OTEL_EMPTY="}
			want := []string{"RUST_BACKTRACE=full", "RUST_LOG=debug", "OTEL_SPAN_FILTER=scp=trace", "OTEL_EXPORTER_OTLP_ENDPOINT=http://collector", "OTEL_EMPTY="}
			if tc.hash != nil {
				want = append(want, "GROUNDHOG_NETWORK_HASH=abcd"+strings.Repeat("00", 30))
			}
			if got := scpdEnv(parent, "debug", tc.hash); !reflect.DeepEqual(got, want) {
				t.Fatalf("environment = %v, want %v", got, want)
			}
		})
	}
}
