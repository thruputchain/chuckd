package xdrutil

import (
	"reflect"
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestMarshal(t *testing.T) {

	m := NewXdrMarshaler()

	x := proto.PublicKey{}
	*x.Ed25519() = [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1}
	var y proto.PublicKey
	err := UnmarshalXdr(&y, m.MarshalXdr(&x).Bytes())
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(x, y) {
		t.Errorf("expecting equality, expecting %v, got %v", x, y)
	}
}
