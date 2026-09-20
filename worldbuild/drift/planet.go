package drift

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/adaralex/strata/server/world/h3x"
)

// Planet is the cell graph at one resolution: every cell, its terrain class,
// centre, and six neighbours by index. Neighbour and edge tables are built
// once and shared by the fifteen solves.
type Planet struct {
	Res    int
	Cells  []h3x.Cell // sorted
	Class  []Class
	Lat    []float32
	Lon    []float32
	Nbr    []int32   // n*6, -1 for none
	EdgeKm []float32 // n*6
	index  map[h3x.Cell]int32
}

// Index returns a cell's position, or -1.
func (p *Planet) Index(c h3x.Cell) int32 {
	if i, ok := p.index[c]; ok {
		return i
	}
	return -1
}

// AllCells enumerates every cell at res via the 122 base cells.
func AllCells(res int) ([]h3x.Cell, error) {
	base, err := h3x.Res0Cells()
	if err != nil {
		return nil, err
	}
	var out []h3x.Cell
	for _, b := range base {
		kids, err := h3x.Children(b, res)
		if err != nil {
			return nil, err
		}
		out = append(out, kids...)
	}
	h3x.SortCells(out)
	return out, nil
}

// BuildPlanet classifies cells and wires the graph. cells may be a subset
// (tests, regional runs): neighbours outside it are simply absent.
func BuildPlanet(res int, cells []h3x.Cell, t *Terrain, progress io.Writer) (*Planet, error) {
	log := func(format string, a ...any) {
		if progress != nil {
			_, _ = fmt.Fprintf(progress, format+"\n", a...)
		}
	}
	start := time.Now()
	n := len(cells)
	p := &Planet{Res: res, Cells: cells, Class: make([]Class, n), Lat: make([]float32, n), Lon: make([]float32, n), index: make(map[h3x.Cell]int32, n)}
	for i, c := range cells {
		p.index[c] = int32(i)
		lat, lon, err := h3x.Center(c)
		if err != nil {
			return nil, err
		}
		p.Lat[i], p.Lon[i] = float32(lat), float32(lon)
		p.Class[i] = t.Grid.At(lat, lon)
	}
	log("planet: %d cells classified in %s", n, time.Since(start).Round(time.Millisecond))

	// Rivers: land-ish cells the centrelines pass through become river.
	edgeKm, _ := h3x.EdgeLengthKm(res)
	riverCells := 0
	for _, ls := range t.Rivers {
		cs, err := h3x.LineCells(ls, res, edgeKm*500) // half an edge, in metres
		if err != nil {
			continue
		}
		for _, c := range cs {
			if i := p.Index(c); i >= 0 {
				switch p.Class[i] {
				case Land, Mountain, Desert:
					p.Class[i] = River
					riverCells++
				}
			}
		}
	}
	log("planet: %d river cells", riverCells)

	// Neighbours and edge lengths; coast is land next to ocean.
	p.Nbr = make([]int32, n*6)
	p.EdgeKm = make([]float32, n*6)
	for i := range p.Nbr {
		p.Nbr[i] = -1
	}
	coast := 0
	for i, c := range cells {
		ring, err := c.GridRing(1)
		if err != nil {
			// Pentagon distortion: fall back to the disk minus self.
			disk, derr := h3x.Disk(c, 1)
			if derr != nil {
				continue
			}
			ring = ring[:0]
			for _, d := range disk {
				if d != c {
					ring = append(ring, d)
				}
			}
		}
		touchesOcean := false
		for k, nc := range ring {
			if k >= 6 {
				break
			}
			j := p.Index(nc)
			p.Nbr[i*6+k] = j
			if j < 0 {
				continue
			}
			p.EdgeKm[i*6+k] = float32(haversineKm(float64(p.Lat[i]), float64(p.Lon[i]), float64(p.Lat[j]), float64(p.Lon[j])))
			if p.Class[j] == Ocean {
				touchesOcean = true
			}
		}
		if touchesOcean && p.Class[i] == Land {
			p.Class[i] = Coast
			coast++
		}
	}
	log("planet: graph wired, %d coast cells, %s", coast, time.Since(start).Round(time.Millisecond))
	return p, nil
}

// ClassCounts tallies cells per class.
func (p *Planet) ClassCounts() map[string]int {
	out := map[string]int{}
	for _, c := range p.Class {
		out[c.String()]++
	}
	return out
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0088
	rad := math.Pi / 180
	dlat := (lat2 - lat1) * rad
	dlon := (lon2 - lon1) * rad
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	return 2 * r * math.Asin(math.Min(1, math.Sqrt(a)))
}
