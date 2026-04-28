package router

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type dynamicUniformRouter struct {
	m          libxdr.XdrMarshaler
	discovery  *discovery.DiscoveryService
	treeFanOut int
}

func NewDynamicUniformRouter(
	ctx context.Context,
	discovery *discovery.DiscoveryService,
	treeFanOut int,
) Router {
	return &dynamicUniformRouter{
		discovery:  discovery,
		m:          libxdr.NewXdrMarshaler(),
		treeFanOut: treeFanOut,
	}
}

func (x *dynamicUniformRouter) Self() proto.ValidatorPublicInfo {
	return x.discovery.Self()
}

func (x *dynamicUniformRouter) OurQuorumSet() proto.SCPQuorumSet {
	return x.discovery.OurQuorumSet()
}

func (x *dynamicUniformRouter) ResolveAddress(vid proto.ValidatorID) (string, error) {
	v, found := x.discovery.LookupValidator(vid)
	if !found {
		return "", fmt.Errorf("validator not found")
	}
	return v.Address, nil
}

func (x *dynamicUniformRouter) ChooseSCPProviders(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	vs := protoutil.SnapshotValidators(x.discovery.CachedSnapshots())
	vs = dedupExcludeSelf(vs, protoutil.PublicVID(x.Self())) // filter out self
	if len(vs) > 0 {
		r := rand.New(rand.NewSource(seed))
		r.Shuffle(len(vs), func(i int, j int) { vs[i], vs[j] = vs[j], vs[i] })
		return vs[:1]
	} else {
		return nil
	}
}

func (x *dynamicUniformRouter) ChooseCatchupProviders(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	vs := protoutil.SnapshotValidators(x.discovery.CachedSnapshots())
	vs = dedupExcludeSelf(vs, protoutil.PublicVID(x.Self())) // filter out self
	if len(vs) == 0 {
		return nil
	}
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(vs), func(i int, j int) { vs[i], vs[j] = vs[j], vs[i] })
	return vs[:1]
}

func (x *dynamicUniformRouter) BuildDogwoodTree(rootID ValidatorID, maxSize int) *dogwoodutil.LinkedDAGNode {
	vs := protoutil.SnapshotValidatorIDs(x.discovery.CachedSnapshots())
	return buildDogwoodTree(x.treeFanOut, rootID, vs, maxSize)
}

// buildDogwoodTree builds a uniform tree of validators rooted at rootID.
// At most maxSize nodes are produced (including the root). vs is consumed
// in the process; rootID is removed from it on entry.
func buildDogwoodTree(fanOut int, rootID ValidatorID, vs map[proto.ValidatorID]bool, maxSize int) *dogwoodutil.LinkedDAGNode {
	delete(vs, rootID)
	root := &dogwoodutil.LinkedDAGNode{Node: rootID}
	if maxSize <= 1 || len(vs) == 0 {
		return root
	}
	remaining := maxSize - 1 // budget for non-root nodes
	ch := make(chan *dogwoodutil.LinkedDAGNode, len(vs)+1)
	ch <- root
	for n := range ch {
		d := min(len(vs), fanOut, remaining)
		if d <= 0 {
			break
		}
		n.Children = make([]*dogwoodutil.LinkedDAGNode, d)
		for i := range n.Children {
			for id := range vs {
				delete(vs, id)
				n.Children[i] = &dogwoodutil.LinkedDAGNode{Node: id}
				ch <- n.Children[i]
				remaining--
				break
			}
		}
		if remaining == 0 || len(vs) == 0 {
			break
		}
	}
	return root
}

func divideRoundUp(x, y int) int {
	if x%y == 0 {
		return x / y
	}
	return (x / y) + 1
}

func dedupExcludeSelf(vs []ValidatorPublicInfo, vid ValidatorID) []ValidatorPublicInfo {
	m := map[ValidatorID]ValidatorPublicInfo{}
	for _, v := range vs {
		m[protoutil.PublicVID(v)] = v
	}
	_, ok := m[vid]
	if !ok {
		panic("cannot find root")
	}
	delete(m, vid)
	dedup := protoutil.ValidatorPublicInfos{}
	for _, v := range m {
		dedup = append(dedup, v)
	}
	dedup.Sort()
	return dedup
}
