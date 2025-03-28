package xdrutil

import (
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

type DbCards []proto.DbCard

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
