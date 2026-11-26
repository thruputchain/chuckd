package xdrutil

import (
	"bytes"
	"testing"
)

func TestAppendCxsUint32(t *testing.T) {
	for _, tc := range []struct {
		n    uint32
		want []byte
	}{
		{0, []byte{0}},
		{62, []byte{62}},
		{63, []byte{63, 63}},
		{255, []byte{63, 255}},
		{256, []byte{127, 0, 1}},
		{0xffff, []byte{127, 0xff, 0xff}},
		{0x10000, []byte{191, 0, 0, 1, 0}},
		{0xffffffff, []byte{191, 0xff, 0xff, 0xff, 0xff}},
	} {
		if got := appendCxsUint32(nil, tc.n); !bytes.Equal(got, tc.want) {
			t.Errorf("appendCxsUint32(%d) = %x, want %x", tc.n, got, tc.want)
		}
	}
}

func TestAppendCxsUint32s(t *testing.T) {
	got := appendCxsUint32s([]byte{0xaa}, []uint32{63, 300, 70000})
	want := []byte{0xaa, 3, 63, 63, 127, 0x2c, 0x01, 191, 0x70, 0x11, 0x01, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("appendCxsUint32s = %x, want %x", got, want)
	}
	if got := appendCxsUint32s(nil, nil); !bytes.Equal(got, []byte{0}) {
		t.Errorf("appendCxsUint32s of no elements = %x, want 00", got)
	}
}
