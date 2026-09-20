package drift

import (
	"math"
	"sort"

	"github.com/adaralex/strata/server/world"
)

// RankByNormalisedCost orders civilizations by cost over base lambda, the
// same ranking the cell builder uses to choose the top four.
func RankByNormalisedCost(costs [world.NumCivs]float64, civs *world.Civs) []uint8 {
	ids := make([]uint8, 0, world.NumCivs)
	for i := range costs {
		if !math.IsInf(costs[i], 1) {
			ids = append(ids, uint8(i))
		}
	}
	sort.Slice(ids, func(a, b int) bool {
		return costs[ids[a]]/civs.Lambda(ids[a], 0) < costs[ids[b]]/civs.Lambda(ids[b], 0)
	})
	return ids
}

// RecordFromCosts packs fifteen costs into a cell record: top four by
// normalised cost, the rest folded into the residual. Shared by the cell
// builder and the query tool so both agree.
func RecordFromCosts(costs [world.NumCivs]float64, civs *world.Civs) world.Record {
	rec := world.EmptyRecord()
	order := RankByNormalisedCost(costs, civs)
	for i := 0; i < world.TopK && i < len(order); i++ {
		rec.Civ[i] = order[i]
		rec.Cost[i] = world.QuantiseCost(costs[order[i]] / civs.CostUnitKm)
	}
	cMin := world.CMinKm(&rec, civs)
	var tail []world.CivWeight
	for _, id := range order[min(world.TopK, len(order)):] {
		tail = append(tail, world.CivWeight{Civ: id, Weight: math.Exp(-costs[id] / civs.LambdaEff(id, 0, cMin))})
	}
	rec.ResidualCost = world.FoldResidual(tail, civs, cMin)
	return rec
}
