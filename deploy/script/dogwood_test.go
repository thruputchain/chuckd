package script

import (
	"context"
	"fmt"
	"testing"

	"github.com/groundhogchain/chuckd/lib/librand"
)

func TestDogwoodLocalScript(t *testing.T) {

	ctx := context.Background()

	def := BenchDogwoodExecDef{
		QUIC:          true,
		BytesPerPay:   300e3,
		NumIterations: 50000,
		MetricsPort:   2222,
		BenchPort:     4000,
	}

	fmt.Println(def.LocalScript(ctx, librand.New(), "chuckd", 3))
}
