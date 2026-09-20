// Command worldbuild runs the offline pipeline of PLAN.md §14 steps 1-4 for
// one OSM extract: classify POIs, bake exclusion zones, rasterise the
// civilization layers into H3 r8 cell records, and write a snapshot.
//
//	worldbuild -osm midi-pyrenees-latest.osm.pbf -out out/midi-pyrenees
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/worldbuild/cells"
	"github.com/adaralex/strata/worldbuild/classify"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "worldbuild:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		osmPath  = flag.String("osm", "", "OSM extract (.osm.pbf or .osm)")
		out      = flag.String("out", "", "snapshot directory to write")
		rulesDir = flag.String("rules", "rules", "rules directory (civs.json, poi_classes.json)")
		dataDir  = flag.String("data", "data", "data directory (cores/, beacons/, lists)")
		bbox     = flag.String("bbox", "", "override cell coverage: minLat,minLon,maxLat,maxLon")
		buildID  = flag.String("build-id", "", "snapshot build id (default: file name and date)")
	)
	flag.Parse()
	if *osmPath == "" || *out == "" {
		flag.Usage()
		return fmt.Errorf("-osm and -out are required")
	}
	start := time.Now()
	civsJSON, err := os.ReadFile(filepath.Join(*rulesDir, "civs.json"))
	if err != nil {
		return err
	}
	civs, err := world.ParseCivs(civsJSON)
	if err != nil {
		return err
	}
	rules, err := classify.LoadRules(filepath.Join(*rulesDir, "poi_classes.json"))
	if err != nil {
		return err
	}
	layers, err := cells.LoadLayers(filepath.Join(*dataDir, "cores"), civs)
	if err != nil {
		return err
	}
	curated, err := cells.LoadCurated(filepath.Join(*dataDir, "beacons"), civs)
	if err != nil {
		return err
	}
	civTags, err := cells.LoadCivTags(filepath.Join(*dataDir, "civ_tags.json"), civs)
	if err != nil {
		return err
	}
	block, err := cells.LoadBlocklist(filepath.Join(*dataDir, "blocklist.geojson"))
	if err != nil {
		return err
	}
	lists, err := classify.LoadLists(*dataDir, rules)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "reading %s\n", *osmPath)
	ex, err := classify.ReadOSM(context.Background(), *osmPath, rules, os.Stderr)
	if err != nil {
		return err
	}
	res := classify.Run(ex, rules, lists)
	fmt.Fprintf(os.Stderr, "classified: %v; excluded %d; exclusion zones %v\n", res.Stats.ByClass, res.Stats.Excluded, res.Stats.ByExclusion)

	bounds := ex.Bounds
	if *bbox != "" {
		bounds, err = parseBBox(*bbox)
		if err != nil {
			return err
		}
	}
	if *buildID == "" {
		*buildID = strings.TrimSuffix(strings.TrimSuffix(filepath.Base(*osmPath), ".pbf"), ".osm") + "-" + time.Now().UTC().Format("20060102")
	}
	snap, st, err := cells.Build(cells.Inputs{
		BuildID: *buildID, Civs: civs, Rules: rules, Layers: layers, Curated: curated, CivTags: civTags,
		Blocklist: block, Classified: res, Bounds: bounds, Progress: os.Stderr,
	})
	if err != nil {
		return err
	}
	if err := snap.Write(*out, civsJSON); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s: %d cells (%d reached), %d excluded r10 cells, %d live POIs, %d beacons in %s\n",
		*out, st.Cells, st.Reached, st.ExclCells, st.LivePOIs, st.Beacons, time.Since(start).Round(time.Millisecond))
	return nil
}

func parseBBox(s string) (classify.BBox, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return classify.BBox{}, fmt.Errorf("bbox wants minLat,minLon,maxLat,maxLon")
	}
	var v [4]float64
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return classify.BBox{}, fmt.Errorf("bbox: %w", err)
		}
		v[i] = f
	}
	return classify.BBox{MinLat: v[0], MinLon: v[1], MaxLat: v[2], MaxLon: v[3]}, nil
}
