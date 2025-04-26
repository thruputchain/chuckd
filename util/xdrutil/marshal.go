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

func UnmarshalXdr(val xdr.XdrType, in []byte) (e error) {
	defer func() {
		switch i := recover().(type) {
		case nil:
		case xdr.XdrError:
			e = i
		default:
			panic(i)
		}
	}()
	val.XdrMarshal(&xdr.XdrIn{In: bytes.NewBuffer(in)}, "")
	return nil
}
