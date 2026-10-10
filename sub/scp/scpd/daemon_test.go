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
		name   string
		hash   *proto.NetworkHash
		parent []string
		want   []string
	}{
		{name: "without hash"},
		{name: "with hash", hash: &hash},
		{name: "service name", parent: []string{"OTEL_SERVICE_NAME=chuckd"}},
		{
			name:   "resource identity removed",
			parent: []string{"OTEL_RESOURCE_ATTRIBUTES=deployment.environment=devnet,service.name=chuckd,service.instance.id=abc,host.role=val%2C1"},
			want:   []string{"OTEL_RESOURCE_ATTRIBUTES=deployment.environment=devnet,host.role=val%2C1"},
		},
		{
			name:   "resource attributes unchanged",
			parent: []string{"OTEL_RESOURCE_ATTRIBUTES=deployment.environment=devnet"},
			want:   []string{"OTEL_RESOURCE_ATTRIBUTES=deployment.environment=devnet"},
		},
		{name: "only resource identity", parent: []string{"OTEL_RESOURCE_ATTRIBUTES=service.name=chuckd, service.instance.id=x"}},
		{
			name:   "empty pairs and repeated identity",
			parent: []string{"OTEL_RESOURCE_ATTRIBUTES=, service.name =chuckd,,host.role=val%2C1,service.name=other, service.instance.id =x, "},
			want:   []string{"OTEL_RESOURCE_ATTRIBUTES=host.role=val%2C1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := []string{"PATH=/bin", "HOME=/home", "OTELX=1", "MY_OTEL_FOO=1", "RUST_LOG=wrong", "OTEL_MALFORMED", "OTEL_SPAN_FILTER=scp=trace", "OTEL_EXPORTER_OTLP_ENDPOINT=http://collector", "OTEL_EMPTY="}
			want := []string{"RUST_BACKTRACE=full", "RUST_LOG=debug", "OTEL_SPAN_FILTER=scp=trace", "OTEL_EXPORTER_OTLP_ENDPOINT=http://collector", "OTEL_EMPTY="}
			parent = append(parent, tc.parent...)
			want = append(want, tc.want...)
			if tc.hash != nil {
				want = append(want, "GROUNDHOG_NETWORK_HASH=abcd"+strings.Repeat("00", 30))
			}
			if got := scpdEnv(parent, "debug", tc.hash); !reflect.DeepEqual(got, want) {
				t.Fatalf("environment = %v, want %v", got, want)
			}
		})
	}
}
