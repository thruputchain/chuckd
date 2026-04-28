package router

import (
	"context"
	"math"
	"strconv"
	"testing"

	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/librand"
	"github.com/groundhogchain/lib-go/must"
)

func generateValidators(ctx context.Context, n int) []ValidatorPublicInfo {
	rr := librand.NewRand()
	vs := make([]ValidatorPublicInfo, n)
	for i := range vs {
		vs[i] = protoutil.MustGenerateIdentity(ctx, rr, strconv.Itoa(i)).Public_info
	}
	return vs
}

func countNodes(t *dogwoodutil.LinkedDAGNode) int {
	if t == nil {
		return 0
	}
	n := 1
	for _, ch := range t.Children {
		n += countNodes(ch)
	}
	return n
}

func TestBasicDogwood(t *testing.T) {

	ctx := context.Background()
	vs := generateValidators(ctx, 7)

	maxFanOut := 3
	tree := buildDogwoodTree(
		maxFanOut,
		protoutil.PublicVID(vs[0]),
		protoutil.ValidatorPublicToIDs(vs[1:]),
		math.MaxInt,
	)

	must.Assert(ctx, len(tree.Children) == 3)
	must.Assert(ctx, len(tree.Children[0].Children) == 3)
	must.Assert(ctx, len(tree.Children[1].Children) == 0)
	must.Assert(ctx, len(tree.Children[2].Children) == 0)
}

func TestBuildDogwoodTree_MaxSizeCapsTree(t *testing.T) {
	ctx := context.Background()
	const total = 100
	vs := generateValidators(ctx, total)
	root := protoutil.PublicVID(vs[0])

	for _, maxSize := range []int{1, 2, 5, 17, 50, total, 1000} {
		ids := protoutil.ValidatorPublicToIDs(vs[1:])
		tree := buildDogwoodTree(4, root, ids, maxSize)

		want := maxSize
		if want > total {
			want = total
		}
		if got := countNodes(tree); got != want {
			t.Errorf("maxSize=%d: tree has %d nodes, want %d", maxSize, got, want)
		}
		if tree.Node != root {
			t.Errorf("maxSize=%d: tree root mismatch", maxSize)
		}
	}
}

func TestBuildDogwoodTree_MaxSizeOne(t *testing.T) {
	ctx := context.Background()
	vs := generateValidators(ctx, 5)
	root := protoutil.PublicVID(vs[0])

	tree := buildDogwoodTree(3, root, protoutil.ValidatorPublicToIDs(vs[1:]), 1)
	must.Assert(ctx, tree.Node == root)
	must.Assert(ctx, len(tree.Children) == 0)
}

func TestBuildDogwoodTree_MaxSizeZeroOrNegative(t *testing.T) {
	ctx := context.Background()
	vs := generateValidators(ctx, 5)
	root := protoutil.PublicVID(vs[0])

	for _, maxSize := range []int{0, -1, -100} {
		ids := protoutil.ValidatorPublicToIDs(vs[1:])
		tree := buildDogwoodTree(3, root, ids, maxSize)
		if tree == nil {
			t.Fatalf("maxSize=%d: got nil tree", maxSize)
		}
		if tree.Node != root {
			t.Fatalf("maxSize=%d: root mismatch", maxSize)
		}
		if len(tree.Children) != 0 {
			t.Fatalf("maxSize=%d: expected no children, got %d", maxSize, len(tree.Children))
		}
	}
}

func TestBuildDogwoodTree_RespectsFanOut(t *testing.T) {
	ctx := context.Background()
	vs := generateValidators(ctx, 100)
	root := protoutil.PublicVID(vs[0])

	const fanOut = 3
	const maxSize = 20
	tree := buildDogwoodTree(fanOut, root, protoutil.ValidatorPublicToIDs(vs[1:]), maxSize)

	must.Assert(ctx, countNodes(tree) == maxSize)

	stack := []*dogwoodutil.LinkedDAGNode{tree}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if len(n.Children) > fanOut {
			t.Fatalf("node has %d children, exceeds fanOut %d", len(n.Children), fanOut)
		}
		stack = append(stack, n.Children...)
	}
}
