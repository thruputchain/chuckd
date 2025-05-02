package xdrutil

import (
	"fmt"
	"reflect"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestRoundtrip(t *testing.T) {
	// x := proto.ValidatorPublicInfo{Address: "1.2.3.4"}
	x := proto.BroadcastChannel{Batch_count: 111}
	// x := proto.Config{
	// 	Spectra_config: &proto.SpectraConfig{
	// 		Partial_ttl_msec: 333,
	// 	},
	// }
	s, err := XdrToJson(&x)
	if err != nil {
		t.Error(err)
	}
	// fmt.Println("XXX", string(s))
	// y := proto.ValidatorPublicInfo{}
	y := proto.BroadcastChannel{}
	// y := proto.Config{}
	err = JsonToXdr(&y, s)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(x, y) {
		t.Errorf("expecting equality")
	}
}

type testNonXdrType struct {
	Bool           bool
	Int32          int32
	Int64          int64
	Unexported     bool
	ByteArray      [3]byte
	SomeXdr        proto.SHA256Hash
	Complex        complex128
	MissingPointer *[]string
	PresentPointer *string
	SliceString    []string
	Bytes          []byte
	EmbeddedXDR    proto.ValidatorPublicInfo
}

func TestAnyToJson(t *testing.T) {
	ss := "prezo"
	v := &testNonXdrType{
		SliceString:    []string{"abc", "def"},
		Bytes:          []byte{0, 1, 2},
		PresentPointer: &ss,
		EmbeddedXDR:    proto.ValidatorPublicInfo{Address: "1.2.3.4"},
	}
	s, err := AnyToJson(v)
	if err != nil {
		t.Error(err)
	}
	expected := `
{
    "Bool": false,
    "Int32": 0,
    "Int64": "0",
    "Unexported": false,
    "ByteArray": "AAAA",
    "SomeXdr": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
    "Complex": go:(0+0i),
    "MissingPointer": null,
    "PresentPointer": "prezo",
    "SliceString": [
        "abc",
        "def"
    ],
    "Bytes": "AAEC",
    "EmbeddedXDR": {
        "Public_key": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
        "Address": "1.2.3.4"
    }
}`
	if string(s) != expected {
		t.Errorf("expecting %q, got %q", expected, string(s))
	}
	fmt.Println(string(s))
}
