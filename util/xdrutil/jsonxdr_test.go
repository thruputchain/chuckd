package xdrutil

import (
	"fmt"
	"reflect"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestUnion(t *testing.T) {
	x := proto.SCPStatement{}
	x.Pledges.Stmt_type = proto.SCP_ST_CONFIRM
	*x.Pledges.Confirm() = proto.SCPStatementConfirm{
		Ballot:          proto.SCPBallot{Counter: 5, Value: proto.SCPValue{1, 2, 3}},
		N_prepared:      1,
		N_commit:        2,
		N_h:             3,
		Quorum_set_hash: proto.SCPHash{1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 3, 4}, // SCPHash
	}
	s, err := XdrToJson(&x)
	if err != nil {
		t.Error(err)
	}
	println(string(s))
	y := proto.SCPStatement{}
	err = JsonToXdr(&y, s)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(x, y) {
		t.Errorf("expecting equality")
	}
}

func TestRoundtrip(t *testing.T) {
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
    "Complex": (0+0i),
    "MissingPointer": <nil>,
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
		t.Errorf("expecting %s, got %s", expected, string(s))
	}
	fmt.Println(string(s))
}
