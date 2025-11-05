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
		TreeFanOut:    5,
		BytesPerPay:   300e3,
		NumIterations: 50000,
		MetricsPort:   2222,
		BenchPort:     4000,
	}
	fmt.Println(def.LocalScript(ctx, librand.NewRand(), "chuckd", 3))
}

func TestDogwoodScriptTwoHogs(t *testing.T) {
	ctx := context.Background()
	def := BenchDogwoodExecDef{
		TreeFanOut:    5,
		BytesPerPay:   300e3, // 300Kb
		NumIterations: 50000,
		MetricsPort:   2222,
		BenchPort:     4000,
	}
	for _, s := range def.HostScripts(ctx, librand.NewRand(), "$1", "hog7", "hog8") {
		fmt.Println(s)
	}
}
