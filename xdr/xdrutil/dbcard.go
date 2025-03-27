package xdrutil

import (
	"sort"

	"github.com/groundhogchain/chuckd/xdr/xdrdef"
)

type DbCards []xdrdef.DbCard

func (x DbCards) Len() int {
	return len(x)
}

func (x DbCards) Less(i, j int) bool {
	if LessUPID(x[i].Upid, x[j].Upid) {
		return true
	}
	return x[i].Address < x[j].Address
}

func (x DbCards) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x DbCards) Sort() {
	sort.Sort(x)
}
