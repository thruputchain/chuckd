package librand

import "math/rand"

func New() *rand.Rand {
	return rand.New(rand.NewSource(rand.Int63()).(rand.Source64))
}
