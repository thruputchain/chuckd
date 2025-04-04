package router

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

func TestBasic(t *testing.T) {

	// generate validator identities
	nv := 7
	vs := make([]ValidatorPublicInfo, nv)
	for i := range vs {
		vs[i] = ValidatorPublicInfo{Address: strconv.Itoa(i)}
	}

	maxFanOut := 3
	treeNodes, _, _ := buildTree(maxFanOut, vs[0], vs[1:])
	for _, tn := range treeNodes {
		h := protoutil.MulticastTreeNodeHash(tn)
		fmt.Println(form.SprintJSON(&h))
		fmt.Println(form.SprintJSON(&tn))
	}
}
