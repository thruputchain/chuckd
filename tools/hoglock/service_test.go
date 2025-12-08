package hoglock

import (
	"reflect"
	"testing"

	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libruntime"
	proto "github.com/groundhogchain/protocol-go"
)

func TestService(t *testing.T) {

	ctx := t.Context()
	liblog.LogLevelDebug()
	libruntime.InstallStackTraceOnInterrupt()

	s := NewService([]string{"h1", "h2", "h3"})

	// lock h1, h2
	a1, err := s.Hoglock_lock(ctx, "x1", proto.HogAsk{Fixed: proto.HogHosts{"h1"}, Variable: 1})
	if err != nil {
		t.Fatal(err)
	}
	liblog.Infof("a1=%v", a1)
	if len(a1) != 1 || a1[0] != "h2" {
		t.Fatalf("a1=%v", a1)
	}

	// lock h3 right away
	a2, err := s.Hoglock_lock(ctx, "x2", proto.HogAsk{Fixed: proto.HogHosts{}, Variable: 1})
	if err != nil {
		t.Fatal(err)
	}
	liblog.Infof("a2=%v", a2)
	if len(a2) != 1 || a2[0] != "h3" {
		t.Fatalf("a2=%v", a2)
	}

	d0 := s.Demand()
	exp0 := WeightedHosts{{Host: "h1", Weight: 1}, {Host: "h2", Weight: 1}, {Host: "h3", Weight: 1}}
	if !reflect.DeepEqual(exp0, d0) {
		t.Fatalf("unexpected demand %v", d0)
	}

	s.releaseLease(ctx, "x1")

	d1 := s.Demand()
	exp1 := WeightedHosts{{Host: "h1", Weight: 0}, {Host: "h2", Weight: 0}, {Host: "h3", Weight: 1}}
	if !reflect.DeepEqual(exp1, d1) {
		t.Fatalf("unexpected demand %v", d1)
	}

}
