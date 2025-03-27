package xdrutil

import (
	"encoding/binary"

	"github.com/groundhogchain/chuckd/xdr/xdrdef"
)

func LessUPID(a, b xdrdef.UPID) bool {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return true
		} else if a[i] > b[i] {
			return false
		}
	}
	return false // a == b
}

func Uint64UPID(i uint64) xdrdef.UPID {
	var u xdrdef.UPID
	binary.LittleEndian.PutUint64(u[:], i)
	return u
}
