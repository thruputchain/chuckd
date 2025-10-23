package scpd

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func TestConfigPrinting(t *testing.T) {
	ctx := context.Background()
	rand := librand.New()
	nv := 3
	vs := make([]proto.ValidatorPrivateInfo, nv)
	ks := make([]proto.ValidatorPublicInfo, nv)
	ids := make([]proto.ValidatorID, nv)
	for i := 0; i < nv; i++ {
		vs[i] = *protoutil.MustGenerateIdentity(ctx, rand, fmt.Sprintf("addr:%d", i))
		ks[i] = vs[i].Public_info
		ids[i] = protoutil.PublicVID(ks[i])
	}
	cfg := proto.SCPConfig{
		Max_timeout_ms:                  1,
		Initial_ballot_timeout_ms:       2,
		Increment_ballot_timeout_ms:     3,
		Initial_nomination_timeout_ms:   4,
		Increment_nomination_timeout_ms: 5,
		Quorum_set: proto.SCPQuorumSet{
			Threshold:  2,
			Validators: protoutil.ValidatorToNodeIDs(ids),
			Inner_sets: nil,
		},
	}
	b, _ := json.MarshalIndent(SCPConfigToShim(cfg), "", "   ")
	println(string(b))
}
