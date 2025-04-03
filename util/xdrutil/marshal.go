package xdrutil

import (
	"bytes"

	"github.com/xdrpp/goxdr/xdr"
)

func MarshalXdr(value xdr.XdrType) []byte {
	var w bytes.Buffer
	out := xdr.XdrOut{Out: &w}
	out.Marshal("", value)
	return w.Bytes()
}
