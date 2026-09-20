// Package cond computes the condition vector of PLAN.md §5 for one cell and
// one epoch: solar phase, moon, local hour and the propagation scalar. Phase
// 0 pins weather to fair; the fields exist so rules can already name them.
//
// Everything is derived from the cell's own coordinates and UTC on the
// server. A device clock is not evidence (PLAN §5, "Server authority").
package cond

import (
	"math"
	"time"
)

// EpochSeconds is the spawn epoch length: fifteen minutes (PLAN §4).
const EpochSeconds = 900

// Phase is the solar phase bucket.
type Phase uint8

// Solar phases. Twilight is civil twilight, sun between 0 and -6 degrees.
const (
	Day Phase = iota
	Twilight
	Night
)

func (p Phase) String() string {
	switch p {
	case Day:
		return "day"
	case Twilight:
		return "twilight"
	default:
		return "night"
	}
}

// Propagation parameters, from rules/spawn.json.
type Propagation struct {
	Day       float64 `json:"day"`
	Twilight  float64 `json:"twilight"`
	Night     float64 `json:"night"`
	MoonBonus float64 `json:"moon_bonus"` // added at night, scaled by illumination
}

// Vector is the condition vector for one cell at one epoch.
type Vector struct {
	Epoch int64     `json:"epoch"`
	At    time.Time `json:"at"` // epoch start, UTC

	SolarAltDeg float64 `json:"solar_alt_deg"`
	Phase       Phase   `json:"-"`
	PhaseName   string  `json:"phase"`
	IsNight     bool    `json:"is_night"`
	MoonIllum   float64 `json:"moon_illumination"`
	// LocalHour is mean solar time at the cell, 0..24. Good enough for the
	// after-midnight dampening of PLAN §11 without a time-zone database.
	LocalHour     float64 `json:"local_hour"`
	AfterMidnight bool    `json:"after_midnight"`
	Propagation   float64 `json:"propagation"`

	// Weather is pinned to fair in phase 0 (PLAN §5 fallback).
	Weather string `json:"weather"`
	Precip  bool   `json:"precip"`
	Storm   bool   `json:"storm"`
	Fog     bool   `json:"fog"`
	Frost   bool   `json:"frost"`
}

// EpochOf returns the epoch index containing t.
func EpochOf(t time.Time) int64 { return t.Unix() / EpochSeconds }

// EpochStart returns the UTC start of an epoch.
func EpochStart(epoch int64) time.Time { return time.Unix(epoch*EpochSeconds, 0).UTC() }

// At computes the vector for a cell centre at the epoch containing t.
func At(lat, lon float64, t time.Time, p Propagation) Vector {
	epoch := EpochOf(t)
	start := EpochStart(epoch)
	v := Vector{Epoch: epoch, At: start, Weather: "fair"}
	v.SolarAltDeg = SolarAltitude(lat, lon, start)
	switch {
	case v.SolarAltDeg > 0:
		v.Phase = Day
	case v.SolarAltDeg > -6:
		v.Phase = Twilight
	default:
		v.Phase = Night
	}
	v.PhaseName = v.Phase.String()
	v.IsNight = v.Phase == Night
	v.MoonIllum = MoonIllumination(start)
	v.LocalHour = math.Mod(float64(start.Hour())+float64(start.Minute())/60+lon/15+48, 24)
	v.AfterMidnight = v.LocalHour < 5
	switch v.Phase {
	case Day:
		v.Propagation = p.Day
	case Twilight:
		v.Propagation = p.Twilight
	default:
		v.Propagation = p.Night + p.MoonBonus*v.MoonIllum
	}
	v.Propagation = math.Max(0, math.Min(1, v.Propagation))
	return v
}

// Digest packs the regional fields into one 64-bit id. Every cell in a
// region under the same sky shares it, so eligible rule sets can be memoised
// per digest (PLAN §5). Per-cell statics are not part of it.
func (v Vector) Digest() uint64 {
	d := uint64(v.Phase) & 0x3
	d |= (uint64(v.MoonIllum*10+0.5) & 0xF) << 2
	if v.AfterMidnight {
		d |= 1 << 6
	}
	d |= (uint64(v.Propagation*20+0.5) & 0x1F) << 7
	if v.Precip {
		d |= 1 << 12
	}
	if v.Storm {
		d |= 1 << 13
	}
	if v.Fog {
		d |= 1 << 14
	}
	if v.Frost {
		d |= 1 << 15
	}
	return d | 0xC0DE<<48 // tag so a zero vector is never a zero digest
}

// SolarAltitude returns the sun's altitude in degrees above the horizon at
// (lat, lon) and time t, using the NOAA low-precision algorithm. Accurate
// to a fraction of a degree, which is far finer than the phase buckets.
func SolarAltitude(lat, lon float64, t time.Time) float64 {
	const rad = math.Pi / 180
	jd := float64(t.Unix())/86400 + 2440587.5
	n := jd - 2451545.0
	L := math.Mod(280.460+0.9856474*n, 360)
	g := math.Mod(357.528+0.9856003*n, 360) * rad
	lambda := (L + 1.915*math.Sin(g) + 0.020*math.Sin(2*g)) * rad
	eps := (23.439 - 0.0000004*n) * rad
	alpha := math.Atan2(math.Cos(eps)*math.Sin(lambda), math.Cos(lambda))
	delta := math.Asin(math.Sin(eps) * math.Sin(lambda))
	gmst := math.Mod(18.697374558+24.06570982441908*n, 24)
	if gmst < 0 {
		gmst += 24
	}
	ha := (gmst*15+lon)*rad - alpha
	la := lat * rad
	alt := math.Asin(math.Sin(la)*math.Sin(delta) + math.Cos(la)*math.Cos(delta)*math.Cos(ha))
	return alt / rad
}

// MoonIllumination returns the illuminated fraction of the moon, 0..1, from
// the mean synodic cycle. Within a few percent, which is all a rule needs.
func MoonIllumination(t time.Time) float64 {
	const synodic = 29.530588853
	ref := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC) // a new moon
	days := t.Sub(ref).Hours() / 24
	phase := days/synodic - math.Floor(days/synodic)
	return (1 - math.Cos(2*math.Pi*phase)) / 2
}
