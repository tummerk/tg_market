package tests

import (
	"math/rand"
	"time"
)

type Randomizer struct {
	Float64 func() float64
	Bool    func() bool
}

func NewRandomizer() Randomizer {
	random := rand.New(rand.NewSource(time.Now().Unix())) //nolint:gosec // for tests

	return Randomizer{
		Float64: random.Float64,
		Bool:    func() bool { return random.Intn(2) == 0 }, //nolint:mnd // skip
	}
}
