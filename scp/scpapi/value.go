package scpapi

import (
	"bytes"
	"sort"
)

type Values []Value

func (vs Values) Len() int {
	return len(vs)
}

func (vs Values) Less(i, j int) bool {
	return bytes.Compare(vs[i], vs[j]) < 0
}

func (vs Values) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs Values) Sort() {
	sort.Sort(vs)
}
