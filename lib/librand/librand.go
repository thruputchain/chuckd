package librand

import "math/rand"

func New() *rand.Rand {
	return rand.New(rand.NewSource(rand.Int63()).(rand.Source64))
}

func NewFromRand(parent *rand.Rand) *rand.Rand {
	return rand.New(rand.NewSource(parent.Int63()))
}
