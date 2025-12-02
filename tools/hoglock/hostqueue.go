package hoglock

import (
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

type WeightedHost struct {
	Host   proto.HogHost
	Weight int
}

func WeightedHostsFromMap(demand map[proto.HogHost]int, excluding proto.HogHosts) WeightedHosts {

	excluded := map[proto.HogHost]bool{}
	for _, x := range excluding {
		excluded[x] = true
	}

	r := make(WeightedHosts, 0, len(demand))
	for h, w := range demand {
		if !excluded[h] {
			r = append(r, WeightedHost{Host: h, Weight: w})
		}
	}
	r.Sort()
	return r
}

type WeightedHosts []WeightedHost

func (x WeightedHosts) ToHosts() proto.HogHosts {
	h := make(proto.HogHosts, len(x))
	for i := range x {
		h[i] = x[i].Host
	}
	return h
}

func (x WeightedHosts) Len() int {
	return len(x)
}

func (x WeightedHosts) Less(i, j int) bool {
	if x[i].Weight < x[j].Weight {
		return true
	}
	if x[i].Weight == x[j].Weight {
		return x[i].Host < x[j].Host // hosts by lex order
	}
	return false
}

func (x WeightedHosts) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x WeightedHosts) Sort() {
	sort.Sort(x)
}
