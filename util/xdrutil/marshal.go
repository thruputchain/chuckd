package xdrutil

import (
	"bytes"

	"github.com/xdrpp/goxdr/xdr"
)

var mpool = NewBufferPool()

func MarshalXdr(value xdr.XdrType) []byte {

	w := mpool.Get()
	defer mpool.Recycle(w)

	out := xdr.XdrOut{Out: w}
	out.Marshal("", value)

	d := make([]byte, w.Len())
	copy(d, w.Bytes())

	return d
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
