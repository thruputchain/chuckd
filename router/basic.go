package router

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"

	"github.com/groundhogchain/chuckd/discovery"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

type basicRouter struct {
	discovery *discovery.DiscoveryService
	maxFanOut int
}

func NewBasicRouter(
	ctx context.Context,
	discovery *discovery.DiscoveryService,
	maxFanOut int,
) Router {
	return &basicRouter{
		discovery: discovery,
		maxFanOut: maxFanOut,
	}
}

func (x *basicRouter) BuildTree(rootID ValidatorID, cid ChannelID) (
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
	treeHash MulticastTreeHash,
) {

	// prepare randomness
	mix := sha256.Sum256(cid[:])
	seed := int64(binary.LittleEndian.Uint64(mix[:8]))
	r := rand.New(rand.NewSource(seed))

	// shuffle validators
	vs := protoutil.SnapshotValidators(x.discovery.CachedSnapshots())
	r.Shuffle(len(vs), func(i int, j int) { vs[i], vs[j] = vs[j], vs[i] })

	// find and remove root
	root, vs, ok := findPopValidator(rootID, vs)
	if !ok {
		panic("missing validator")
	}

	return buildTree(x.maxFanOut, root, vs)
}

func buildTree(
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
		for _, ch := range children {
			chop := min(m, len(unused))
			chTreeNodes, _, chTreeHash := buildTree(maxFanOut, ch, unused[:chop])
			unused = unused[chop:]
			treeNodes = append(treeNodes, chTreeNodes...)
			treeNodes[0].Children = append(treeNodes[0].Children, chTreeHash)
		}
	}
	treeHash = protoutil.MulticastTreeNodeHash(treeNodes[0])

	return treeNodes, children, treeHash
}

func divideRoundUp(x, y int) int {
	if x%y == 0 {
		return x / y
	}
	return (x / y) + 1
}

func findPopValidator(vid ValidatorID, vs []ValidatorPublicInfo) (ValidatorPublicInfo, []ValidatorPublicInfo, bool) {

	for i, v := range vs {
		if protoutil.ValidatorIDEqual(vid, protoutil.ValidatorPublicKeyToID(v.Public_key)) {
			vs[i] = vs[len(vs)-1]
			vs = vs[:len(vs)-1]
			return v, vs, true
		}
	}
	return ValidatorPublicInfo{}, nil, false
}
