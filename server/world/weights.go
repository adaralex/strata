package world

import "math"

// CivWeight is one entry of a derived weight vector.
type CivWeight struct {
	Civ    uint8
	Weight float64
}

// Weights is the derived, normalised weight vector of a cell under a
// propagation scalar, plus the residual share and the purity scalar.
type Weights struct {
	Top      []CivWeight // TopK at most, descending, empty slots skipped
	Residual float64     // share of the folded tail
	Purity   float64     // max weight / sum, 1.0 at a core
	// Reached is false when no layer reaches the cell at all. Track 2's
	// acceptance criterion is that this never happens on the planet.
	Reached bool
}

// Derive turns stored costs into weights:
//
//	lambda_i = lambda_base_i * (1 + k_i * propagation)
//	w_i      = exp(-cost_i / lambda_i)
//	weights  = normalise(w)
//	purity   = max(w) / sum(w)
func Derive(rec *Record, civs *Civs, propagation float64) Weights {
	propagation = math.Max(0, math.Min(1, propagation))
	unit := civs.CostUnitKm
	var raw [TopK]float64
	var sum, maxW float64
	for i := 0; i < TopK; i++ {
		if rec.Civ[i] == NoCiv || rec.Cost[i] == CostUnreached {
			continue
		}
		cost := float64(rec.Cost[i]) * unit
		raw[i] = math.Exp(-cost / civs.Lambda(rec.Civ[i], propagation))
		sum += raw[i]
		if raw[i] > maxW {
			maxW = raw[i]
		}
	}
	var res float64
	if rec.ResidualCost != CostUnreached {
		res = math.Exp(-float64(rec.ResidualCost) * unit / civs.MeanLambda(propagation))
		sum += res
	}
	out := Weights{Reached: sum > 0}
	if !out.Reached {
		return out
	}
	for i := 0; i < TopK; i++ {
		if raw[i] > 0 {
			out.Top = append(out.Top, CivWeight{Civ: rec.Civ[i], Weight: raw[i] / sum})
		}
	}
	out.Residual = res / sum
	out.Purity = maxW / sum
	return out
}

// FoldResidual turns the weights of the civilizations that did not make the
// top K into one pseudo-cost such that exp(-residual/meanLambda) equals their
// summed baseline weight. Used by the builder; returns CostUnreached when the
// tail is empty.
func FoldResidual(tail []CivWeight, civs *Civs) uint16 {
	var s float64
	for _, t := range tail {
		s += t.Weight
	}
	if s <= 0 {
		return CostUnreached
	}
	cost := -math.Log(s) * civs.MeanLambda(0) / civs.CostUnitKm
	return QuantiseCost(cost)
}

// QuantiseCost clamps a cost in cost units to the u16 range, keeping 0xFFFF
// for "unreached".
func QuantiseCost(cost float64) uint16 {
	if math.IsInf(cost, 1) || math.IsNaN(cost) {
		return CostUnreached
	}
	if cost < 0 {
		cost = 0
	}
	if cost >= CostUnreached-1 {
		return CostUnreached - 1
	}
	return uint16(cost + 0.5)
}

func log2(x float64) float64 { return math.Log2(x) }
func pow2(x float64) float64 { return math.Pow(2, x) }
