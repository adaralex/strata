package cond

import (
	"math"
	"testing"
	"time"
)

var p = Propagation{Day: 0.15, Twilight: 0.5, Night: 0.85, MoonBonus: 0.1}

func TestSolarAltitudeToulouse(t *testing.T) {
	// Solar noon at Toulouse (lon 1.44) on the June solstice is ~11:54 UTC;
	// altitude = 90 - 43.6 + 23.44 = 69.8 degrees.
	noon := time.Date(2026, 6, 21, 11, 54, 0, 0, time.UTC)
	alt := SolarAltitude(43.6, 1.44, noon)
	if math.Abs(alt-69.8) > 1.5 {
		t.Fatalf("solstice noon altitude %.1f, want ~69.8", alt)
	}
	midnight := time.Date(2026, 12, 21, 1, 0, 0, 0, time.UTC)
	if alt := SolarAltitude(43.6, 1.44, midnight); alt > -50 {
		t.Fatalf("winter night altitude %.1f should be deep below the horizon", alt)
	}
	// Civil twilight on 21 June: sunrise about 04:20 UTC, so 04:00 is twilight.
	dawn := time.Date(2026, 6, 21, 4, 0, 0, 0, time.UTC)
	if alt := SolarAltitude(43.6, 1.44, dawn); alt > 0 || alt < -6 {
		t.Fatalf("dawn altitude %.1f should be in civil twilight", alt)
	}
}

func TestVectorPhases(t *testing.T) {
	day := At(43.5365, 1.3444, time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC), p)
	if day.Phase != Day || day.IsNight || day.Propagation != 0.15 {
		t.Fatalf("noon: %+v", day)
	}
	night := At(43.5365, 1.3444, time.Date(2026, 9, 20, 23, 30, 0, 0, time.UTC), p)
	if night.Phase != Night || !night.IsNight || night.Propagation < 0.85 || night.AfterMidnight {
		t.Fatalf("23:30 UTC: %+v", night)
	}
	late := At(43.5365, 1.3444, time.Date(2026, 9, 21, 1, 0, 0, 0, time.UTC), p)
	if !late.AfterMidnight {
		t.Fatalf("01:00 UTC is after local midnight at lon 1.3: %+v", late)
	}
	if day.Digest() == night.Digest() {
		t.Fatal("day and night must have different digests")
	}
	if day.Digest() == 0 || day.Digest()>>48 != 0xC0DE {
		t.Fatalf("digest tag: %x", day.Digest())
	}
}

func TestEpoch(t *testing.T) {
	a := time.Date(2026, 9, 20, 12, 7, 0, 0, time.UTC)
	b := time.Date(2026, 9, 20, 12, 14, 59, 0, time.UTC)
	c := time.Date(2026, 9, 20, 12, 15, 0, 0, time.UTC)
	if EpochOf(a) != EpochOf(b) || EpochOf(b) == EpochOf(c) {
		t.Fatal("epochs are 15-minute buckets")
	}
	if !EpochStart(EpochOf(a)).Equal(time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("epoch start")
	}
}

func TestMoon(t *testing.T) {
	// 2026-03-03 is a full moon; 2026-03-19 a new moon (to within a day).
	if f := MoonIllumination(time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)); f < 0.95 {
		t.Fatalf("full moon illumination %.2f", f)
	}
	if f := MoonIllumination(time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)); f > 0.05 {
		t.Fatalf("new moon illumination %.2f", f)
	}
}
