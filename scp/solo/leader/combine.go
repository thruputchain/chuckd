package leader

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/scp/scpapi"
	proto "github.com/groundhogchain/protocol-go"
)

type RandomChoiceCombiner struct {
	lk   sync.Mutex
	rand *rand.Rand
}

func NewRandomChoiceCombiner(ctx context.Context, rand *rand.Rand) *RandomChoiceCombiner {
	return &RandomChoiceCombiner{rand: rand}
}

func (x *RandomChoiceCombiner) CombineCandidates(ctx context.Context, slotIndex scpapi.Slot, values []scpapi.Value) scpapi.Value {
	x.lk.Lock()
	defer x.lk.Unlock()
	return values[x.rand.Intn(len(values))]
}

func (x *RandomChoiceCombiner) ValidateValue(ctx context.Context, slotIndex scpapi.Slot, value scpapi.Value, nomination bool) scpapi.ValidationLevel {
	return proto.SoloMaybeValidValue
}

func (x *RandomChoiceCombiner) ExtractValidValue(ctx context.Context, slotIndex scpapi.Slot, value scpapi.Value) *scpapi.Value {
	return nil
}

func (x *RandomChoiceCombiner) ExternalizedValue(ctx context.Context, slotIndex scpapi.Slot, value scpapi.Value) {
}
