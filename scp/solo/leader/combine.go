package leader

import (
	"context"
	"math/rand"
	"sync"

	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type UnionCombiner struct{}

func (UnionCombiner) CombineCandidates(ctx context.Context, slotIndex scpapi.Slot, values []scpapi.Value) scpapi.Value {
	ss := make([]proto.SoloStatement, len(values))
	for i := range values {
		var s proto.SoloStatement
		xdrutil.UnmarshalXdr(&s, values[i])
		ss[i] = s
	}
	return xdrutil.MarshalXdr(&proto.SoloStatementSet{Statements: ss})
}

func (UnionCombiner) ValidateValue(ctx context.Context, slotIndex scpapi.Slot, value scpapi.Value, nomination bool) scpapi.ValidationLevel {
	return proto.SoloMaybeValidValue
}

func (UnionCombiner) ExtractValidValue(ctx context.Context, slotIndex scpapi.Slot, value scpapi.Value) *scpapi.Value {
	return nil
}

func (UnionCombiner) ExternalizedValue(ctx context.Context, slotIndex scpapi.Slot, value scpapi.Value) {
}

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
