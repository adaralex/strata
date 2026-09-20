package main

import (
	"fmt"
	"os"
	"time"

	"github.com/adaralex/strata/server/spawn"
	"github.com/adaralex/strata/server/world"
)

// devSecret is used when STRATA_SECRET is unset, so the CLI and a worldd
// started the same way agree. Never deploy with it.
const devSecret = "strata-phase0-dev-secret-not-for-production"

func spawns(w *world.World, lat, lon float64, at, rulesDir string) error {
	t := time.Now()
	if at != "" {
		var err error
		if t, err = time.Parse(time.RFC3339, at); err != nil {
			return fmt.Errorf("-t: %w", err)
		}
	}
	secret := os.Getenv("STRATA_SECRET")
	if len(secret) < 16 {
		secret = devSecret
	}
	content, err := spawn.LoadContent(rulesDir, w.Civs)
	if err != nil {
		return err
	}
	sp, err := spawn.New(w, content, []byte(secret))
	if err != nil {
		return err
	}
	list, v, err := sp.Near(lat, lon, t)
	if err != nil {
		return err
	}
	fmt.Printf("epoch %d (%s), %s, sun %.1f°, moon %.0f%%, propagation %.2f, digest %x\n",
		v.Epoch, v.At.Format(time.RFC3339), v.PhaseName, v.SolarAltDeg, v.MoonIllum*100, v.Propagation, v.Digest())
	fmt.Printf("%d spawns in the cell and its neighbours, nearest first:\n", len(list))
	for _, s := range list {
		hy := ""
		if s.Secondary != "" {
			hy = " (hybrid with " + s.Secondary + ")"
		}
		fmt.Printf("%6.0f m  %-10s %-13s %-34s %-10s x%.2f  %s%s\n", sp.DistanceM(lat, lon, s), s.Civ, s.TierName, s.Name, s.Authenticity, s.ValueMul, s.ID, hy)
	}
	return nil
}
