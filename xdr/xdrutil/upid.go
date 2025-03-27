package xdrutil

import "github.com/groundhogchain/chuckd/xdr/xdrdef"

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
