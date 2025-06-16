package xdrutil

import (
	"bytes"

	"github.com/xdrpp/goxdr/xdr"
)

type XdrMarshaler struct {
	bpool *BufferPool
}

func NewXdrMarshaler() XdrMarshaler {
	return XdrMarshaler{bpool: NewBufferPool()}
}

func (x XdrMarshaler) MarshalXdr(value xdr.XdrType) []byte {

	w := x.bpool.Get()
	defer x.bpool.Recycle(w)

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
