// Command drift runs phase 0 track 2 (PLAN.md §4 "The drift field"):
//
//	drift fetch -out out/ne                          download the Natural Earth layers
//	drift solve -ne out/ne -out out/drift/drift.bin  classify the planet and run the fifteen solves
//	drift verify -field out/drift/drift.bin          count unreached cells (acceptance: none)
//	drift query -field out/drift/drift.bin LAT LON   costs and weights at a point, day and night
package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/worldbuild/cells"
	"github.com/adaralex/strata/worldbuild/drift"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "drift:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: drift fetch|solve|verify|query")
	}
	switch args[0] {
	case "fetch":
		fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
		out := fs.String("out", "out/ne", "directory for the Natural Earth GeoJSON files")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return fetch(*out)
	case "solve":
		fs := flag.NewFlagSet("solve", flag.ContinueOnError)
		ne := fs.String("ne", "out/ne", "Natural Earth directory")
		out := fs.String("out", "out/drift/drift.bin", "field to write")
		rulesDir := fs.String("rules", "rules", "rules directory")
		dataDir := fs.String("data", "data", "data directory")
		res := fs.Int("res", 0, "H3 resolution (default from rules/drift.json)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return solve(*ne, *out, *rulesDir, *dataDir, *res)
	case "verify":
		fs := flag.NewFlagSet("verify", flag.ContinueOnError)
		field := fs.String("field", "out/drift/drift.bin", "field to check")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		return verify(*field)
	case "query":
		fs := flag.NewFlagSet("query", flag.ContinueOnError)
		field := fs.String("field", "out/drift/drift.bin", "field to read")
		rulesDir := fs.String("rules", "rules", "rules directory")
		// Coordinates may be negative, which the flag parser would read as
		// flags: pull numeric tokens out first.
		var nums []float64
		var rest []string
		for _, a := range args[1:] {
			if f, err := strconv.ParseFloat(a, 64); err == nil {
				nums = append(nums, f)
			} else {
				rest = append(rest, a)
			}
		}
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if len(nums) != 2 {
			return fmt.Errorf("query wants LAT LON")
		}
		lat, lon := nums[0], nums[1]
		return query(*field, *rulesDir, lat, lon)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func fetch(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	for _, name := range drift.Sources {
		path := filepath.Join(dir, name+".geojson")
		if st, err := os.Stat(path); err == nil && st.Size() > 0 {
			fmt.Fprintf(os.Stderr, "have %s (%d bytes)\n", path, st.Size())
			continue
		}
		url := drift.SourceURL(name)
		fmt.Fprintf(os.Stderr, "fetching %s\n", url)
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			_ = resp.Body.Close()
			return fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
		}
		f, err := os.Create(path)
		if err != nil {
			_ = resp.Body.Close()
			return err
		}
		n, err := io.Copy(f, resp.Body)
		_ = resp.Body.Close()
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", path, n)
	}
	fmt.Fprintln(os.Stderr, "Natural Earth data is public domain: https://www.naturalearthdata.com/about/terms-of-use/")
	return nil
}

func solve(neDir, out, rulesDir, dataDir string, res int) error {
	start := time.Now()
	civs, err := world.LoadCivs(filepath.Join(rulesDir, "civs.json"))
	if err != nil {
		return err
	}
	rules, err := drift.LoadRules(filepath.Join(rulesDir, "drift.json"))
	if err != nil {
		return err
	}
	if res == 0 {
		res = rules.Resolution
	}
	layers, err := cells.LoadLayers(filepath.Join(dataDir, "cores"), civs)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "terrain: rasterising Natural Earth at %.3f°\n", rules.RasterStepDeg)
	terrain, err := drift.LoadTerrain(neDir, rules.RasterStepDeg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "terrain: %v in %s\n", terrain.Stats, time.Since(start).Round(time.Millisecond))
	all, err := drift.AllCells(res)
	if err != nil {
		return err
	}
	planet, err := drift.BuildPlanet(res, all, terrain, os.Stderr)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "planet: classes %v\n", planet.ClassCounts())
	field, err := drift.Solve(planet, rules, civs, layers, os.Stderr)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := field.Write(out); err != nil {
		return err
	}
	pairs, cellsAny := field.Unreached()
	fmt.Fprintf(os.Stderr, "wrote %s: r%d, %d cells, %d unreached (cell, civ) pairs in %d cells, %s\n", out, res, len(field.Keys), pairs, cellsAny, time.Since(start).Round(time.Millisecond))
	return nil
}

func verify(path string) error {
	f, err := drift.Read(path)
	if err != nil {
		return err
	}
	pairs, cellsAny := f.Unreached()
	fmt.Printf("field %s: r%d, %d cells, rules v%d\n", path, f.Res, len(f.Keys), f.RulesVersion)
	if pairs > 0 {
		return fmt.Errorf("ACCEPTANCE FAILED: %d (cell, civ) pairs unreached in %d cells", pairs, cellsAny)
	}
	fmt.Println("every cell reaches all fifteen civilizations: nowhere on the planet returns zero weight")
	return nil
}

func query(path, rulesDir string, lat, lon float64) error {
	f, err := drift.Read(path)
	if err != nil {
		return err
	}
	civs, err := world.LoadCivs(filepath.Join(rulesDir, "civs.json"))
	if err != nil {
		return err
	}
	costs, ok := f.At(lat, lon)
	if !ok {
		return fmt.Errorf("point outside the field")
	}
	rec := drift.RecordFromCosts(costs, civs)
	fmt.Printf("%.4f, %.4f  (r%d field)\n", lat, lon, f.Res)
	fmt.Printf("%-10s %9s   %-16s %-16s\n", "civ", "cost km", "weight day", "weight night")
	day := world.Derive(&rec, civs, 0)
	night := world.Derive(&rec, civs, 1)
	dw, nw := map[uint8]float64{}, map[uint8]float64{}
	for _, cw := range day.Top {
		dw[cw.Civ] = cw.Weight
	}
	for _, cw := range night.Top {
		nw[cw.Civ] = cw.Weight
	}
	order := drift.RankByNormalisedCost(costs, civs)
	for _, id := range order {
		mark := ""
		if _, top := dw[id]; !top {
			mark = "  (residual)"
		}
		fmt.Printf("%-10s %9.0f   %-16.3f %-16.3f%s\n", civs.List[id].Key, costs[id], dw[id], nw[id], mark)
	}
	fmt.Printf("purity     day %.3f  night %.3f;  residual share day %.3f night %.3f\n", day.Purity, night.Purity, day.Residual, night.Residual)
	return nil
}
