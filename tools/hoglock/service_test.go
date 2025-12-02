package hoglock

import (
	"testing"

	"github.com/groundhogchain/chuckd/lib/liblog"
	proto "github.com/groundhogchain/protocol-go"
)

func TestService(t *testing.T) {

	ctx := t.Context()
	liblog.LogLevelDebug()

	s := NewService([]string{"h1", "h2", "h3"})

	a1, err := s.Hoglock_lock(ctx, "x1", proto.HogAsk{Fixed: proto.HogHosts{"h1"}, Variable: 1})
	if err != nil {
		t.Fatal(err)
	}
	liblog.Infof("a1=%v", a1)
	if len(a1) != 1 || a1[0] != "h2" {
		t.Fatalf("a1=%v", a1)
	}

}
