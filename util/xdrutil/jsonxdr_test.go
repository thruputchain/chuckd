package xdrutil

import (
	"fmt"
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

type testNonXdrType struct {
	Primitive   int32
	Unexported  bool
	ByteArray   [3]byte
	SomeXdr     proto.SHA256Hash
	Complex     complex128
	Pointer     *[]string
	SliceString []string
}

func TestAnyToJson(t *testing.T) {
	v := &testNonXdrType{
		SliceString: []string{"abc", "def"},
	}
	s, err := AnyToJson(v)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(s))
}
