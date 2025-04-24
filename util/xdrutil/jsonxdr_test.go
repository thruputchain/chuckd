package xdrutil

import (
	"reflect"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestXdrToJson(t *testing.T) {
	x := proto.ValidatorPublicInfo{Address: "1.2.3.4"}
	s, err := XdrToJson(&x)
	if err != nil {
		t.Error(err)
	}
	y := proto.ValidatorPublicInfo{}
	err = JsonToXdr(&y, s)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(x, y) {
		t.Errorf("expecting equality")
	}
}
