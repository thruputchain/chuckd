package xdrutil

import (
	"encoding/binary"

	proto "github.com/groundhogchain/protocol-go"
)

func LessUPID(a, b proto.UPID) bool {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return true
		} else if a[i] > b[i] {
			return false
		}
	}
	return false // a == b
}

func Uint64UPID(i uint64) proto.UPID {
	var u proto.UPID
	binary.LittleEndian.PutUint64(u[:], i)
	return u
}
