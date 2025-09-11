package xdrutil

import (
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

type ShardCards []proto.ShardCard

func (x ShardCards) Len() int {
	return len(x)
}

func (x ShardCards) Less(i, j int) bool {
	if LessUPID(x[i].Upid, x[j].Upid) {
		return true
	}
	if x[i].Address_for_coord < x[j].Address_for_coord {
		return true
	}
	return x[i].Address_for_exec < x[j].Address_for_exec
}

func (x ShardCards) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x ShardCards) Sort() {
	sort.Sort(x)
}
