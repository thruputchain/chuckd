package xdrutil

import (
	"crypto/sha256"

	"github.com/xdrpp/goxdr/xdr"
)

func HashXdr(m XdrMarshaler, value xdr.XdrType) [32]byte {
	bb := m.MarshalXdr(value)
	defer bb.Recycle()
	return sha256.Sum256(bb.Bytes())
}
