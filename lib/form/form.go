package form

import (
	"context"
	"encoding/json"
	"io"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/xdrpp/goxdr/xdr"
)

type Form interface {
}

type Forms []Form

func ToForms[T Form](xs []T) Forms {
	r := make(Forms, len(xs))
	for i, x := range xs {
		r[i] = x
	}
	return r
}

type Map = map[string]Form

type None struct{}

func SprintJSON(form Form) string {
	switch x := form.(type) {
	case xdr.XdrAggregate:
		b, err := xdrutil.XdrToJson(x)
		if err != nil {
			panic(err)
		}
		return string(b)
	default:
		data, err := json.MarshalIndent(form, "", "   ")
		if err != nil {
			panic(err)
		}
		return string(data)
	}
}

func SprintJSONBytes(form Form) []byte {
	return []byte(SprintJSON(form))
}

func Encode[F Form](ctx context.Context, w io.Writer, f F) error {
	return json.NewEncoder(w).Encode(f)
}

func Decode[F Form](ctx context.Context, r io.Reader) (form F, err error) {
	err = json.NewDecoder(r).Decode(&form)
	return form, err
}

func DecodeInto(ctx context.Context, r io.Reader, into Form) error {
	return json.NewDecoder(r).Decode(into)
}

func EncodeBytes[F Form](ctx context.Context, form F) ([]byte, error) {
	return json.MarshalIndent(form, "", "   ")
}

func DecodeBytes[F Form](ctx context.Context, data []byte) (form F, err error) {
	err = json.Unmarshal(data, &form)
	return form, err
}

func DecodeBytesInto(ctx context.Context, data []byte, into Form) error {
	return json.Unmarshal(data, into)
}
