package xdrutil

import (
	"fmt"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestXdrToText(t *testing.T) {
	// x := proto.BroadcastChannel{Batch_count: 111}
	x := proto.Config{
		Router_config: &proto.RouterConfig{
			Tree_fan_out: 33,
		},
	}
	s, err := XdrToText(&x)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(s)
}

func TestAnyToText(t *testing.T) {
	ss := "prezo"
	x := &testNonXdrType{
		SliceString:    []string{"abc", "def"},
		Bytes:          []byte{0, 1, 2},
		PresentPointer: &ss,
		EmbeddedXDR:    proto.ValidatorPublicInfo{Address: "1.2.3.4"},
	}
	s, err := AnyToText(x)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(s)
}
