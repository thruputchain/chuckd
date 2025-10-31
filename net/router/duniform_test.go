package router

import (
	"context"
	"strconv"
	"testing"

	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

func TestBasicDogwood(t *testing.T) {

	ctx := context.Background()
	rr := librand.NewRand()

	// generate validator identities
	nv := 7
	vs := make([]ValidatorPublicInfo, nv)
	for i := range vs {
		vs[i] = protoutil.MustGenerateIdentity(ctx, rr, strconv.Itoa(i)).Public_info
	}

	maxFanOut := 3
	tree := buildDogwoodTree(
		maxFanOut,
		protoutil.PublicVID(vs[0]),
		protoutil.ValidatorPublicToIDs(vs[1:]),
	)

	must.Assert(ctx, len(tree.Children) == 3)
	must.Assert(ctx, len(tree.Children[0].Children) == 3)
	must.Assert(ctx, len(tree.Children[1].Children) == 0)
	must.Assert(ctx, len(tree.Children[2].Children) == 0)
}
