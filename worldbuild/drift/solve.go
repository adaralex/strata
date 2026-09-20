package drift

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
	"github.com/adaralex/strata/worldbuild/cells"
)

// Rules is rules/drift.json.
type Rules struct {
	Version       int                `json:"version"`
	Resolution    int                `json:"resolution"`
	RasterStepDeg float64            `json:"raster_step_deg"`
	UnitCostKm    map[string]float64 `json:"unit_cost_km"`
	Maritime      struct {
		CoastMul float64 `json:"coast_mul"`
		OceanMul float64 `json:"ocean_mul"`
	} `json:"maritime"`
	Landlocked struct {
		CoastMul float64 `json:"coast_mul"`
		OceanMul float64 `json:"ocean_mul"`
	} `json:"landlocked"`
	MaxCostKm float64 `json:"max_cost_km"`
}

// LoadRules reads rules/drift.json.
func LoadRules(path string) (*Rules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Rules
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, name := range ClassNames {
		if _, ok := r.UnitCostKm[name]; !ok {
			return nil, fmt.Errorf("%s: unit_cost_km missing %q", path, name)
		}
	}
	if r.Resolution < 3 || r.Resolution > 7 {
		return nil, fmt.Errorf("%s: resolution %d outside 3..7", path, r.Resolution)
	}
	if r.RasterStepDeg <= 0 {
		r.RasterStepDeg = 0.05
	}
	if r.MaxCostKm <= 0 {
		r.MaxCostKm = 65000
	}
	return &r, nil
}

// UnitCosts returns the per-class unit cost for one civilization.
func (r *Rules) UnitCosts(civ world.Civ) [numClasses]float32 {
	var u [numClasses]float32
	for i, name := range ClassNames {
		u[i] = float32(r.UnitCostKm[name])
	}
	coastMul, oceanMul := r.Landlocked.CoastMul, r.Landlocked.OceanMul
	if civ.Maritime {
		coastMul, oceanMul = r.Maritime.CoastMul, r.Maritime.OceanMul
	}
	u[Coast] *= float32(coastMul)
	u[Ocean] *= float32(oceanMul * civ.OceanMul)
	return u
}

// Source is a seed cell with its starting cost.
type Source struct {
	Idx    int32
	CostKm float32
}

// SourcesFor rasterises a civilization's layers onto the planet.
func SourcesFor(p *Planet, layers []cells.Layer, civ uint8) ([]Source, error) {
	edgeKm, _ := h3x.EdgeLengthKm(p.Res)
	best := map[int32]float32{}
	add := func(cs []h3x.Cell, cost float32) {
		for _, c := range cs {
			if i := p.Index(c); i >= 0 {
				if old, ok := best[i]; !ok || cost < old {
					best[i] = cost
				}
			}
		}
	}
	for i := range layers {
		l := &layers[i]
		if l.Civ != civ {
			continue
		}
		cost := float32(l.CostKm)
		if l.Line != nil {
			cs, err := h3x.LineCells(l.Line, p.Res, edgeKm*500)
			if err != nil {
				return nil, err
			}
			add(cs, cost)
			continue
		}
		for _, poly := range l.Multi {
			cs, err := h3x.PolyfillOverlapping(poly, p.Res)
			if err != nil {
				return nil, err
			}
			add(cs, cost)
			for _, pt := range poly[0] {
				if c, err := h3x.FromPoint(pt, p.Res); err == nil {
					add([]h3x.Cell{c}, cost)
				}
			}
		}
	}
	out := make([]Source, 0, len(best))
	for i, c := range best {
		out = append(out, Source{Idx: i, CostKm: c})
	}
	return out, nil
}

type item struct {
	d float32
	i int32
}
type pq []item

func (q pq) Len() int           { return len(q) }
func (q pq) Less(a, b int) bool { return q[a].d < q[b].d }
func (q pq) Swap(a, b int)      { q[a], q[b] = q[b], q[a] }
func (q *pq) Push(x any)        { *q = append(*q, x.(item)) }
func (q *pq) Pop() any          { old := *q; it := old[len(old)-1]; *q = old[:len(old)-1]; return it }

// Dijkstra runs one multi-source solve. dist[i] is +Inf where unreached.
func Dijkstra(p *Planet, sources []Source, unit [numClasses]float32) []float32 {
	n := len(p.Cells)
	dist := make([]float32, n)
	for i := range dist {
		dist[i] = float32(math.Inf(1))
	}
	q := make(pq, 0, len(sources))
	for _, s := range sources {
		if s.CostKm < dist[s.Idx] {
			dist[s.Idx] = s.CostKm
			q = append(q, item{s.CostKm, s.Idx})
		}
	}
	heap.Init(&q)
	for q.Len() > 0 {
		it := heap.Pop(&q).(item)
		if it.d > dist[it.i] {
			continue
		}
		ui := unit[p.Class[it.i]]
		for k := 0; k < 6; k++ {
			j := p.Nbr[int(it.i)*6+k]
			if j < 0 {
				continue
			}
			w := p.EdgeKm[int(it.i)*6+k] * (ui + unit[p.Class[j]]) / 2
			if nd := it.d + w; nd < dist[j] {
				dist[j] = nd
				heap.Push(&q, item{nd, j})
			}
		}
	}
	return dist
}

// Solve runs the fifteen solves in parallel and packs the field.
func Solve(p *Planet, r *Rules, civs *world.Civs, layers []cells.Layer, progress io.Writer) (*Field, error) {
	log := func(format string, a ...any) {
		if progress != nil {
			_, _ = fmt.Fprintf(progress, format+"\n", a...)
		}
	}
	n := len(p.Cells)
	f := &Field{Res: p.Res, RulesVersion: r.Version, Keys: make([]uint64, n), Costs: make([]uint16, n*world.NumCivs)}
	for i, c := range p.Cells {
		f.Keys[i] = uint64(c)
	}
	type result struct {
		civ  uint8
		dist []float32
		src  int
		err  error
	}
	results := make(chan result, world.NumCivs)
	sem := make(chan struct{}, max(1, runtime.NumCPU()))
	var wg sync.WaitGroup
	for id := 0; id < world.NumCivs; id++ {
		wg.Add(1)
		go func(civ uint8) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			start := time.Now()
			src, err := SourcesFor(p, layers, civ)
			if err != nil {
				results <- result{civ: civ, err: err}
				return
			}
			if len(src) == 0 {
				results <- result{civ: civ, err: fmt.Errorf("%s has no source cells: add a layer under data/cores", civs.List[civ].Key)}
				return
			}
			dist := Dijkstra(p, src, r.UnitCosts(civs.List[civ]))
			log("solve: %-10s %6d sources, %s", civs.List[civ].Key, len(src), time.Since(start).Round(time.Millisecond))
			results <- result{civ: civ, dist: dist, src: len(src)}
		}(uint8(id))
	}
	wg.Wait()
	close(results)
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		for i, d := range res.dist {
			f.Costs[i*world.NumCivs+int(res.civ)] = quantise(float64(d), r.MaxCostKm)
		}
	}
	return f, nil
}

func quantise(costKm, maxKm float64) uint16 {
	if math.IsInf(costKm, 1) || math.IsNaN(costKm) {
		return world.CostUnreached
	}
	if costKm > maxKm {
		costKm = maxKm
	}
	if costKm >= world.CostUnreached-1 {
		return world.CostUnreached - 1
	}
	return uint16(costKm + 0.5)
}
