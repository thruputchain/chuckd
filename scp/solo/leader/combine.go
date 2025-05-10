package leader

import (
	"context"

	"github.com/groundhogchain/chuckd/scp/api"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type UnionCombiner struct{}

func (UnionCombiner) CombineCandidates(ctx context.Context, slotIndex api.Slot, values []api.Value) api.Value {
	ss := make([]proto.SoloStatement, len(values))
	for i := range values {
		var s proto.SoloStatement
		xdrutil.UnmarshalXdr(&s, values[i])
		ss[i] = s
	}
	return xdrutil.MarshalXdr(&proto.SoloStatementSet{Statements: ss})
}

func (UnionCombiner) ValidateValue(ctx context.Context, slotIndex api.Slot, value api.Value, nomination bool) api.ValidationLevel {
	return proto.SoloMaybeValidValue
}

func (UnionCombiner) ExtractValidValue(ctx context.Context, slotIndex api.Slot, value api.Value) *api.Value {
	return nil
}

func (UnionCombiner) OnExternalizedValue(ctx context.Context, slotIndex api.Slot, value api.Value) {}
