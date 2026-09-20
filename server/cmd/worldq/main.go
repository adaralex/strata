// Command worldq queries a world snapshot from the shell.
//
//	worldq -snapshot out/midi-pyrenees lookup 43.5365 1.3444 [-p 0.5]
//	worldq -snapshot out/midi-pyrenees nearby 43.5365 1.3444 [-radius 3000]
//	worldq -snapshot out/midi-pyrenees spawns 43.5365 1.3444 [-t 2026-09-20T22:00:00Z] [-rules rules]
//	worldq -snapshot out/midi-pyrenees bench
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/adaralex/strata/server/world"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "worldq:", err)
		os.Exit(1)
	}
}

func run() error {
	snapDir := flag.String("snapshot", "", "snapshot directory")
	flag.Parse()
	if *snapDir == "" || flag.NArg() == 0 {
		flag.Usage()
		return fmt.Errorf("usage: worldq -snapshot DIR lookup|nearby|bench")
	}
	t0 := time.Now()
	w, err := world.Load(*snapDir)
	if err != nil {
		return err
	}
	loadTime := time.Since(t0)
	args := flag.Args()
	switch args[0] {
	case "lookup":
		fs := flag.NewFlagSet("lookup", flag.ContinueOnError)
		prop := fs.Float64("p", 0, "propagation scalar 0..1 (PLAN §5)")
		lat, lon, err := latLon(fs, args[1:])
		if err != nil {
			return err
		}
		return lookup(w, lat, lon, *prop)
	case "nearby":
		fs := flag.NewFlagSet("nearby", flag.ContinueOnError)
		radius := fs.Float64("radius", 3000, "radius in metres")
		lat, lon, err := latLon(fs, args[1:])
		if err != nil {
			return err
		}
		return nearby(w, lat, lon, *radius)
	case "spawns":
		fs := flag.NewFlagSet("spawns", flag.ContinueOnError)
		at := fs.String("t", "", "RFC3339 time (default now)")
		rulesDir := fs.String("rules", "rules", "rules directory")
		lat, lon, err := latLon(fs, args[1:])
		if err != nil {
			return err
		}
		return spawns(w, lat, lon, *at, *rulesDir)
	case "bench":
		return bench(w, loadTime)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func latLon(fs *flag.FlagSet, args []string) (float64, float64, error) {
	if len(args) < 2 {
		return 0, 0, fmt.Errorf("need LAT LON")
	}
	lat, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return 0, 0, err
	}
	lon, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return 0, 0, err
	}
	if err := fs.Parse(args[2:]); err != nil {
		return 0, 0, err
	}
	return lat, lon, nil
}

func lookup(w *world.World, lat, lon, prop float64) error {
	t0 := time.Now()
	res, err := w.Lookup(lat, lon, prop)
	dt := time.Since(t0)
	if err != nil {
		return err
	}
	fmt.Printf("cell        %s (r8)\n", res.Cell)
	if !res.Found {
		fmt.Println("outside the snapshot")
		return nil
	}
	rec := &res.Record
	fmt.Printf("propagation %.2f\n", prop)
	if !res.Weights.Reached {
		fmt.Println("weights     unreached: no layer covers this cell (track 2 fills these)")
	}
	for i, cw := range res.Weights.Top {
		fmt.Printf("  %-10s %5.3f   cost %5d km\n", w.CivName(cw.Civ), cw.Weight, rec.Cost[i])
	}
	if res.Weights.Residual > 0 {
		fmt.Printf("  %-10s %5.3f   cost %5d km\n", "(residual)", res.Weights.Residual, rec.ResidualCost)
	}
	fmt.Printf("purity      %.3f\n", res.Weights.Purity)
	fmt.Printf("services    %v\n", res.Services)
	fmt.Printf("density     %d (~%.0f POIs in k-ring 1), urban band %d\n", rec.POIDensity, world.DensityCount(rec.POIDensity), rec.UrbanBand())
	fmt.Printf("beacon      grade %d, %d adjacent", rec.BeaconGrade, rec.BeaconN)
	for _, h := range res.Beacons {
		fmt.Printf("; IN RANGE %s (%s, grade %d, %.0f m, %s)", h.Beacon.Name, h.Beacon.Ref, h.Beacon.Grade, h.DistanceM, civsLabel(h.Beacon))
	}
	fmt.Println()
	fmt.Printf("exclusion   %d/49 r10 children in the spawn raster", rec.ExclChildren)
	if res.ExcludedCell {
		fmt.Printf("; this r10 cell claimed by %s", res.ExcludedCellBy)
	}
	fmt.Println()
	if res.Excluded {
		fmt.Printf("            NO INTERACTION: inside %s %s %q (+%.0f m buffer)\n", res.Zone.Kind, res.Zone.Ref, res.Zone.Name, res.Zone.BufferM)
	} else {
		fmt.Println("            interaction allowed here")
	}
	fmt.Printf("terrain     water band %d, wild %v, coast %v\n", rec.WaterBand(), rec.IsWild(), rec.TerrainFlags&world.TerrainCoast != 0)
	fmt.Printf("lookup took %s\n", dt)
	return nil
}

func nearby(w *world.World, lat, lon, radius float64) error {
	for _, n := range w.Nearby(lat, lon, radius) {
		switch n.Kind {
		case "beacon":
			fmt.Printf("%6.0f m  BEACON grade %d  %-40s %s  %s\n", n.DistanceM, n.Beacon.Grade, n.Beacon.Name, n.Beacon.Ref, civsLabel(n.Beacon))
		default:
			fmt.Printf("%6.0f m  %-11s %-22s %-40s %s\n", n.DistanceM, n.POI.Class, n.POI.Subclass, n.POI.Name, n.POI.Ref)
		}
	}
	return nil
}

func bench(w *world.World, loadTime time.Duration) error {
	fmt.Printf("snapshot %s: %d cells, %d beacons, %d POIs, %d excluded r10 cells, loaded in %s\n",
		w.Snap.BuildID, len(w.Snap.Keys), len(w.Snap.Beacons), len(w.Snap.POIs), len(w.Snap.Excl), loadTime)
	if len(w.Snap.Keys) == 0 {
		return nil
	}
	const n = 200000
	t0 := time.Now()
	for i := 0; i < n; i++ {
		c := w.Snap.Keys[i%len(w.Snap.Keys)]
		// Use the cell's own centre so every query hits.
		lat, lon, err := centre(c)
		if err != nil {
			return err
		}
		if _, err := w.Lookup(lat, lon, 0.3); err != nil {
			return err
		}
	}
	per := time.Since(t0) / n
	fmt.Printf("%d lookups, %s each (acceptance: under 5 ms)\n", n, per)
	return nil
}
