package router

import (
	"context"
	"strconv"
	"testing"

	"github.com/groundhogchain/chuckd/runtime"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
)

func TestBasic(t *testing.T) {

	ctx := context.Background()
	runtime.SetRootSeed(0)
	rr := runtime.NewRandFromRoot()
	m := xdrutil.NewXdrMarshaler()

	// generate validator identities
	nv := 7
	vs := make([]ValidatorPublicInfo, nv)
	for i := range vs {
		vs[i] = protoutil.MustGenerateIdentity(ctx, rr, strconv.Itoa(i)).Public_info
	}

	maxFanOut := 3
	treeNodes, _, treeHash := buildRootedTree(m, maxFanOut, vs[0], vs[1:])
	_ = treeNodes
	// for _, tn := range treeNodes {
	// 	h := protoutil.MulticastTreeNodeHash(tn)
	// 	fmt.Println(protoutil.EncodeBase64(h[:]))
	// 	fmt.Println(form.SprintJSON(&tn))
	// }
	// fmt.Println(protoutil.EncodeBase64(treeHash[:]))
	if protoutil.EncodeBase64(treeHash[:]) != "xiw065z4TuLdFmeshjXMH8yFuw7dtl4E1tyPB0qvcr8=" {
		t.Errorf("incorrect tree")
	}
}
