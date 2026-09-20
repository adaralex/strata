package main

import (
	"fmt"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
)

func centre(key uint64) (float64, float64, error) {
	return h3x.Center(h3x.Cell(key))
}

func civsLabel(b *world.Beacon) string {
	if len(b.Civs) == 0 {
		return "civs: inherits soil (" + b.Source + ")"
	}
	return fmt.Sprintf("civs %v (%s)", b.Civs, b.Source)
}
