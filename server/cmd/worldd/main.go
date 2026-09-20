// Command worldd is the phase 0 world service: lookups, spawns and collapses
// over HTTP for the walk test (PLAN.md §13, track 4). Stateless except for
// the in-memory claims and speed-gate fixes of spawn.MemoryState.
//
//	STRATA_SECRET=... worldd -snapshot out/midi-pyrenees -rules rules -addr :8080
//
//	GET  /healthz
//	GET  /v0/lookup?lat=&lon=[&p=]        weights, purity, services, beacons, exclusion
//	GET  /v0/spawns?lat=&lon=             conditions and the spawns of the cell and its neighbours
//	POST /v0/collapse                     {"spawn_id","cell","epoch","lat","lon"}; header X-Strata-Device
//
// X-Strata-Device is an anonymous device id the client generates once. It is
// not identity (non-negotiable 1); accounts arrive in phase 1.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/adaralex/strata/server/cond"
	"github.com/adaralex/strata/server/spawn"
	"github.com/adaralex/strata/server/world"
)

func main() {
	snapDir := flag.String("snapshot", "", "snapshot directory")
	rulesDir := flag.String("rules", "rules", "rules directory (spawn.json, bestiary.json, items.json)")
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()
	if *snapDir == "" {
		flag.Usage()
		os.Exit(2)
	}
	secret := []byte(os.Getenv("STRATA_SECRET"))
	if len(secret) < 16 {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			log.Fatal(err)
		}
		secret = []byte(hex.EncodeToString(b))
		log.Printf("WARNING: STRATA_SECRET unset or short; using a random secret. Spawns will change on restart and differ from any other server.")
	}
	srv, err := newServer(*snapDir, *rulesDir, secret)
	if err != nil {
		log.Fatal(err)
	}
	hs := &http.Server{
		Addr:              *addr,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	log.Printf("worldd: snapshot %s (%d cells, %d beacons), listening on %s", srv.world.Snap.BuildID, len(srv.world.Snap.Keys), len(srv.world.Snap.Beacons), *addr)
	log.Fatal(hs.ListenAndServe())
}

type server struct {
	world   *world.World
	spawner *spawn.Spawner
	state   *spawn.MemoryState
}

func newServer(snapDir, rulesDir string, secret []byte) (*server, error) {
	w, err := world.Load(snapDir)
	if err != nil {
		return nil, err
	}
	content, err := spawn.LoadContent(rulesDir, w.Civs)
	if err != nil {
		return nil, err
	}
	sp, err := spawn.New(w, content, secret)
	if err != nil {
		return nil, err
	}
	return &server{world: w, spawner: sp, state: spawn.NewMemoryState()}, nil
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "build": s.world.Snap.BuildID})
	})
	mux.HandleFunc("GET /v0/lookup", s.handleLookup)
	mux.HandleFunc("GET /v0/spawns", s.handleSpawns)
	mux.HandleFunc("POST /v0/collapse", s.handleCollapse)
	return logging(mux)
}

type civWeight struct {
	Civ    string  `json:"civ"`
	Weight float64 `json:"weight"`
}

type beaconOut struct {
	Ref       string             `json:"ref"`
	Name      string             `json:"name"`
	Grade     uint8              `json:"grade"`
	Civs      map[string]float64 `json:"civs,omitempty"`
	Source    string             `json:"civ_source"`
	DistanceM float64            `json:"distance_m"`
}

type lookupOut struct {
	Cell         string      `json:"cell"`
	Found        bool        `json:"found"`
	Propagation  float64     `json:"propagation"`
	Weights      []civWeight `json:"weights"`
	Residual     float64     `json:"residual"`
	Purity       float64     `json:"purity"`
	Services     []string    `json:"services"`
	Beacons      []beaconOut `json:"beacons"`
	BeaconGrade  uint8       `json:"beacon_grade"`
	Excluded     bool        `json:"excluded"`
	Zone         *world.Zone `json:"zone,omitempty"`
	ExcludedCell bool        `json:"excluded_cell"`
	Wild         bool        `json:"wild"`
	WaterBand    uint8       `json:"water_band"`
	UrbanBand    uint8       `json:"urban_band"`
	Cond         cond.Vector `json:"cond"`
}

func (s *server) lookup(lat, lon float64, at time.Time, propOverride *float64) (lookupOut, error) {
	cell, err := s.spawner.CellOf(lat, lon)
	if err != nil {
		return lookupOut{}, err
	}
	v, err := s.spawner.Conditions(cell, at)
	if err != nil {
		return lookupOut{}, err
	}
	prop := v.Propagation
	if propOverride != nil {
		prop = *propOverride
	}
	res, err := s.world.Lookup(lat, lon, prop)
	if err != nil {
		return lookupOut{}, err
	}
	out := lookupOut{Cell: res.Cell.String(), Found: res.Found, Propagation: prop, Cond: v, Services: []string{}, Beacons: []beaconOut{}, Weights: []civWeight{}}
	if !res.Found {
		return out, nil
	}
	for _, cw := range res.Weights.Top {
		out.Weights = append(out.Weights, civWeight{Civ: s.world.CivName(cw.Civ), Weight: round3(cw.Weight)})
	}
	out.Residual = round3(res.Weights.Residual)
	out.Purity = round3(res.Weights.Purity)
	if res.Services != nil {
		out.Services = res.Services
	}
	for _, h := range res.Beacons {
		out.Beacons = append(out.Beacons, beaconOut{Ref: h.Beacon.Ref, Name: h.Beacon.Name, Grade: h.Beacon.Grade, Civs: h.Beacon.Civs, Source: h.Beacon.Source, DistanceM: round3(h.DistanceM)})
	}
	out.BeaconGrade = res.Record.BeaconGrade
	out.Excluded, out.Zone, out.ExcludedCell = res.Excluded, res.Zone, res.ExcludedCell
	out.Wild, out.WaterBand, out.UrbanBand = res.Record.IsWild(), res.Record.WaterBand(), res.Record.UrbanBand()
	return out, nil
}

func (s *server) handleLookup(w http.ResponseWriter, r *http.Request) {
	lat, lon, ok := latLon(w, r)
	if !ok {
		return
	}
	var prop *float64
	if p := r.URL.Query().Get("p"); p != "" {
		f, err := strconv.ParseFloat(p, 64)
		if err != nil || f < 0 || f > 1 {
			writeErr(w, 400, "p must be 0..1")
			return
		}
		prop = &f
	}
	out, err := s.lookup(lat, lon, time.Now(), prop)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, out)
}

type spawnOut struct {
	spawn.Spawn
	DistanceM float64 `json:"distance_m"`
}

type spawnsOut struct {
	Cond       cond.Vector `json:"cond"`
	Epoch      int64       `json:"epoch"`
	EpochEnds  int64       `json:"epoch_ends"`
	SenseM     float64     `json:"sense_range_m"`
	SpeedGated bool        `json:"speed_gated"`
	Spawns     []spawnOut  `json:"spawns"`
}

func (s *server) handleSpawns(w http.ResponseWriter, r *http.Request) {
	lat, lon, ok := latLon(w, r)
	if !ok {
		return
	}
	now := time.Now()
	sps, v, err := s.spawner.Near(lat, lon, now)
	if err != nil {
		writeErr(w, 500, err.Error())
		return
	}
	out := spawnsOut{Cond: v, Epoch: v.Epoch, EpochEnds: cond.EpochStart(v.Epoch + 1).Unix(), SenseM: s.spawner.Content.Rules.Placement.SenseRangeM, Spawns: []spawnOut{}}
	if dev := r.Header.Get("X-Strata-Device"); dev != "" {
		sg := s.spawner.Content.Rules.SpeedGate
		out.SpeedGated = s.state.Observe(dev, lat, lon, now, sg.MaxSpeedKmh, sg.MinIntervalS)
	}
	for _, sp := range sps {
		d := s.spawner.DistanceM(lat, lon, sp)
		out.Spawns = append(out.Spawns, spawnOut{Spawn: sp, DistanceM: round3(d)})
	}
	writeJSON(w, 200, out)
}

type collapseIn struct {
	SpawnID string  `json:"spawn_id"`
	Cell    string  `json:"cell"`
	Epoch   int64   `json:"epoch"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

func (s *server) handleCollapse(w http.ResponseWriter, r *http.Request) {
	dev := r.Header.Get("X-Strata-Device")
	if len(dev) < 8 || len(dev) > 128 {
		writeErr(w, 400, "X-Strata-Device header required (8..128 chars)")
		return
	}
	var in collapseIn
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in); err != nil {
		writeErr(w, 400, "bad json: "+err.Error())
		return
	}
	res, err := s.spawner.Collapse(spawn.CollapseRequest{Device: dev, SpawnID: in.SpawnID, Cell: in.Cell, Epoch: in.Epoch, Lat: in.Lat, Lon: in.Lon, At: time.Now()}, s.state)
	if err != nil {
		writeErr(w, collapseStatus(err), err.Error())
		return
	}
	writeJSON(w, 200, res)
}

func collapseStatus(err error) int {
	switch {
	case errors.Is(err, spawn.ErrBadCell), errors.Is(err, spawn.ErrEpochAhead):
		return 400
	case errors.Is(err, spawn.ErrNotFound), errors.Is(err, spawn.ErrNoWorld):
		return 404
	case errors.Is(err, spawn.ErrExpired):
		return 410
	case errors.Is(err, spawn.ErrClaimed):
		return 409
	case errors.Is(err, spawn.ErrTooFar), errors.Is(err, spawn.ErrExcluded), errors.Is(err, spawn.ErrSpeedGate):
		return 403
	default:
		return 500
	}
}

func latLon(w http.ResponseWriter, r *http.Request) (float64, float64, bool) {
	q := r.URL.Query()
	lat, err1 := strconv.ParseFloat(q.Get("lat"), 64)
	lon, err2 := strconv.ParseFloat(q.Get("lon"), 64)
	if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		writeErr(w, 400, "lat and lon are required decimal degrees")
		return 0, 0, false
	}
	return lat, lon, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func round3(f float64) float64 { return float64(int(f*1000+0.5)) / 1000 }

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(t).Round(time.Microsecond))
	})
}
