package xdrutil

import (
	"bytes"

	"github.com/groundhogchain/lib-go/libmem"
	"github.com/xdrpp/goxdr/xdr"
)

type XdrMarshaler struct {
	bpool *libmem.BufferPool
}

func NewXdrMarshaler() XdrMarshaler {
	return XdrMarshaler{bpool: libmem.NewBufferPool()}
}

func (x XdrMarshaler) MarshalXdr(value xdr.XdrType) libmem.BorrowedBuffer {

	bb := x.bpool.Get()

	out := xdr.XdrOut{Out: bb}
	out.Marshal("", value)

	return bb
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
