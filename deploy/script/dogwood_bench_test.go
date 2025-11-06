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

func TestDogwoodScript300k(t *testing.T) {
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

func TestDogwoodScript30m(t *testing.T) {
	ctx := context.Background()
	def := BenchDogwoodExecDef{
		TreeFanOut:    5,
		BytesPerPay:   30e6, // 30Mb
		NumIterations: 1000,
		MetricsPort:   2222,
		BenchPort:     4000,
	}
	for _, s := range def.HostScripts(ctx, librand.NewRand(), "$1", "hog6", "hog7", "hog8") {
		fmt.Println(s)
	}
}

func TestDogwoodScript300m(t *testing.T) {
	ctx := context.Background()
	def := BenchDogwoodExecDef{
		TreeFanOut:    5,
		BytesPerPay:   300e6, // 300Mb
		NumIterations: 1000,
		MetricsPort:   2222,
		BenchPort:     4000,
	}
	for _, s := range def.HostScripts(ctx, librand.NewRand(), "$1", "hog6", "hog7", "hog8") {
		fmt.Println(s)
	}
}
