package router

import (
	"strconv"
	"testing"

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
	treeNodes, _, treeHash := buildRootedTree(maxFanOut, vs[0], vs[1:])
	_ = treeNodes
	// for _, tn := range treeNodes {
	// 	h := protoutil.MulticastTreeNodeHash(tn)
	// 	fmt.Println(protoutil.EncodeBase64(h[:]))
	// 	fmt.Println(form.SprintJSON(&tn))
	// }
	// fmt.Println(protoutil.EncodeBase64(treeHash[:]))
	if protoutil.EncodeBase64(treeHash[:]) != "VuppSEinD6tIEdvNVwF5GugJlBv5c0bTj2lPWBQvnh0=" {
		t.Errorf("incorrect tree")
	}
}
