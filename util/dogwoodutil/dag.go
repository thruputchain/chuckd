package dogwoodutil

import (
	"fmt"

	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// BuildLinkedDAG builds the smallest DAG of LinkedTreeNodes encoded by the
// given treeNodes. Nodes with identical hashes (i.e. identical subtrees) are
// represented by a single shared *LinkedTreeNode. The first node must be the
// root and all nodes must be part of the same DAG. Duplicate entries (same
// hash) in treeNodes are tolerated since they encode the same node.
func BuildLinkedDAG(m libxdr.XdrMarshaler, treeNodes proto.DogwoodTreeNodes) (*LinkedDAGNode, error) {

	if len(treeNodes) == 0 {
		return nil, fmt.Errorf("no tree nodes")
	}
	rootHash := TreeNodeHash(m, treeNodes[0])

	hs := map[proto.DogwoodTreeHash]proto.DogwoodTreeNode{}
	for _, treeNode := range treeNodes {
		h := TreeNodeHash(m, treeNode)
		hs[h] = treeNode
	}

	return buildLinkedDAG(rootHash, hs)
}

func buildLinkedDAG(
	root proto.DogwoodTreeHash,
	hs map[proto.DogwoodTreeHash]proto.DogwoodTreeNode,
) (*LinkedDAGNode, error) {

	// linked caches the LinkedTreeNode for each hash so that all parents
	// referencing the same hash share the same node. Presence in the cache
	// also marks a hash as already scheduled for processing, preventing
	// redundant traversal of shared subDAGs.
	linked := map[proto.DogwoodTreeHash]*LinkedDAGNode{}

	// makeLinked returns the LinkedTreeNode for h, creating it if needed.
	// The bool return indicates whether the node was already present in the
	// cache (and therefore already scheduled or processed).
	makeLinked := func(h proto.DogwoodTreeHash) (*LinkedDAGNode, []proto.DogwoodTreeHash, bool, error) {
		if existing, ok := linked[h]; ok {
			return existing, nil, true, nil
		}
		node, ok := hs[h]
		if !ok {
			return nil, nil, false, fmt.Errorf("missing tree node for hash %x", h)
		}
		l := &LinkedDAGNode{Node: node.Node}
		linked[h] = l
		return l, node.Children, false, nil
	}

	type frame struct {
		linked   *LinkedDAGNode
		children []proto.DogwoodTreeHash
	}

	rootLinked, rootChildren, _, err := makeLinked(root)
	if err != nil {
		return nil, err
	}
	stack := []frame{{linked: rootLinked, children: rootChildren}}

	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		f.linked.Children = make([]*LinkedDAGNode, len(f.children))
		for i, ch := range f.children {
			child, grandChildren, cached, err := makeLinked(ch)
			if err != nil {
				return nil, err
			}
			f.linked.Children[i] = child
			if !cached {
				stack = append(stack, frame{linked: child, children: grandChildren})
			}
		}
	}

	return rootLinked, nil
}
