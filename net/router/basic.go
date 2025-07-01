package router

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/discovery"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type basicRouter struct {
	m          xdrutil.XdrMarshaler
	discovery  *discovery.DiscoveryService
	treeFanOut int
}

func NewBasicRouter(
	ctx context.Context,
	discovery *discovery.DiscoveryService,
	treeFanOut int,
) Router {
	return &basicRouter{
		discovery:  discovery,
		m:          xdrutil.NewXdrMarshaler(),
		treeFanOut: treeFanOut,
	}
}

func (x *basicRouter) Self() proto.ValidatorPublicInfo {
	return x.discovery.Self()
}

func (x *basicRouter) ChooseCatchupProviders(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	vs := protoutil.SnapshotValidators(x.discovery.CachedSnapshots())
	_, vs = dedupFilterOutValidators(vs, protoutil.VIDOfPublic(x.Self())) // filter out self
	must.Assertf(ctx, len(vs) > 0, "empty quorum")
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(vs), func(i int, j int) { vs[i], vs[j] = vs[j], vs[i] })
	return vs[:1]
}

func (x *basicRouter) BuildDogwoodTree(rootID ValidatorID) *dogwoodutil.LinkedTreeNode {
	vs := protoutil.SnapshotValidatorIDs(x.discovery.CachedSnapshots())
	return buildDogwoodTree(x.treeFanOut, rootID, vs)
}

func buildDogwoodTree(fanOut int, rootID ValidatorID, vs map[proto.ValidatorID]bool) *dogwoodutil.LinkedTreeNode {
	delete(vs, rootID)
	root := &dogwoodutil.LinkedTreeNode{Node: rootID}
	ch := make(chan *dogwoodutil.LinkedTreeNode, len(vs)+1)
	ch <- root
	for n := range ch {
		d := min(len(vs), fanOut)
		n.Children = make([]*dogwoodutil.LinkedTreeNode, d)
		for i := range n.Children {
			for id := range vs {
				delete(vs, id)
				n.Children[i] = &dogwoodutil.LinkedTreeNode{Node: id}
				ch <- n.Children[i]
				break
			}
		}
		if len(vs) == 0 {
			break
		}
	}
	return root
}

func (x *basicRouter) BuildTree(rootID ValidatorID, cid ChannelID) (
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
	treeHash MulticastTreeHash,
) {
	vs := protoutil.SnapshotValidators(x.discovery.CachedSnapshots())
	return buildTree(x.m, x.treeFanOut, vs, rootID, cid)
}

// Info about rootID must be available inside vs.
func buildTree(m xdrutil.XdrMarshaler, maxFanOut int, vs []proto.ValidatorPublicInfo, rootID ValidatorID, cid ChannelID) (
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
	treeHash MulticastTreeHash,
) {

	// prepare randomness
	mix := sha256.Sum256(cid[:])
	seed := int64(binary.LittleEndian.Uint64(mix[:8]))
	r := rand.New(rand.NewSource(seed))

	// shuffle validators
	root, vs := dedupFilterOutValidators(vs, rootID)
	r.Shuffle(len(vs), func(i int, j int) { vs[i], vs[j] = vs[j], vs[i] })

	return buildRootedTree(m, maxFanOut, root, vs)
}

func buildRootedTree(
	xm xdrutil.XdrMarshaler,
	maxFanOut int,
	root ValidatorPublicInfo,
	unused []ValidatorPublicInfo,
) (
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
	treeHash MulticastTreeHash,
) {

	// determine children
	k := min(len(unused), maxFanOut)
	children = unused[:k]
	unused = unused[k:]

	// build children's subtrees
	treeNodes = make(MulticastTreeNodes, 1)
	treeNodes[0].Node = root

	if len(children) > 0 {
		m := divideRoundUp(len(unused), len(children))
		for _, child := range children {
			chop := min(m, len(unused))
			childTreeNodes, _, childTreeHash := buildRootedTree(xm, maxFanOut, child, unused[:chop])
			unused = unused[chop:]
			treeNodes = append(treeNodes, childTreeNodes...)
			treeNodes[0].Children = append(treeNodes[0].Children, childTreeHash)
		}
	}
	treeHash = protoutil.MulticastTreeNodeHash(xm, treeNodes[0])

	return treeNodes, children, treeHash
}

func divideRoundUp(x, y int) int {
	if x%y == 0 {
		return x / y
	}
	return (x / y) + 1
}

func dedupFilterOutValidators(vs []ValidatorPublicInfo, vid ValidatorID) (ValidatorPublicInfo, []ValidatorPublicInfo) {
	m := map[ValidatorID]ValidatorPublicInfo{}
	for _, v := range vs {
		m[protoutil.VIDOfPublic(v)] = v
	}
	found, ok := m[vid]
	if !ok {
		panic("cannot find root")
	}
	delete(m, vid)
	dedup := protoutil.ValidatorPublicInfos{}
	for _, v := range m {
		dedup = append(dedup, v)
	}
	dedup.Sort()
	return found, dedup
}
