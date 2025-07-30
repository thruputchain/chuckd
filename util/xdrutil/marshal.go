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

type BorrowedBytes struct {
	buffer *bytes.Buffer
	pool   *BufferPool
}

func (x BorrowedBytes) Bytes() []byte {
	return x.buffer.Bytes()
}

func (x BorrowedBytes) Recycle() {
	x.pool.Recycle(x.buffer)
}

func (x XdrMarshaler) MarshalXdr(value xdr.XdrType) BorrowedBytes {

	w := x.bpool.Get()

	out := xdr.XdrOut{Out: w}
	out.Marshal("", value)

	return BorrowedBytes{buffer: w, pool: x.bpool}
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
