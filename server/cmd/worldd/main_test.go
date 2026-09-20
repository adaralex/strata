package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/adaralex/strata/server/internal/fixture"
	"github.com/adaralex/strata/server/spawn"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, err := newServer(fixture.Snapshot(t), filepath.Join(fixture.Root(), "rules"), []byte("phase-zero-test-secret-0123456789"))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, url string, v any) int {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Strata-Device", "test-device-1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			t.Fatal(err)
		}
	}
	return resp.StatusCode
}

func TestLookupAndSpawns(t *testing.T) {
	ts := testServer(t)
	var lk lookupOut
	if st := get(t, ts.URL+"/v0/lookup?lat=43.5365&lon=1.3444", &lk); st != 200 {
		t.Fatalf("lookup status %d", st)
	}
	if !lk.Found || len(lk.Weights) != 3 || lk.Weights[0].Civ != "hallstatt" || lk.Purity == 0 {
		t.Fatalf("lookup: %+v", lk)
	}
	if st := get(t, ts.URL+"/v0/lookup?lat=91&lon=0", nil); st != 400 {
		t.Fatalf("bad lat status %d", st)
	}
	var sp spawnsOut
	if st := get(t, ts.URL+"/v0/spawns?lat=43.5365&lon=1.3444", &sp); st != 200 {
		t.Fatalf("spawns status %d", st)
	}
	if len(sp.Spawns) == 0 || sp.Epoch == 0 || sp.Cond.PhaseName == "" {
		t.Fatalf("spawns: %+v", sp)
	}
	for i := 1; i < len(sp.Spawns); i++ {
		if sp.Spawns[i].DistanceM < sp.Spawns[i-1].DistanceM {
			t.Fatal("spawns must be nearest first")
		}
	}
	var again spawnsOut
	get(t, ts.URL+"/v0/spawns?lat=43.5365&lon=1.3444", &again)
	if len(again.Spawns) != len(sp.Spawns) || again.Spawns[0].ID != sp.Spawns[0].ID {
		t.Fatal("two requests in one epoch must agree")
	}
}

func TestCollapseEndpoint(t *testing.T) {
	ts := testServer(t)
	var sp spawnsOut
	get(t, ts.URL+"/v0/spawns?lat=43.5365&lon=1.3444", &sp)
	var target *spawnOut
	for i := range sp.Spawns {
		var lk lookupOut
		get(t, ts.URL+"/v0/lookup?lat="+ftoa(sp.Spawns[i].Lat)+"&lon="+ftoa(sp.Spawns[i].Lon), &lk)
		if !lk.Excluded {
			target = &sp.Spawns[i]
			break
		}
	}
	if target == nil {
		t.Skip("all fixture spawns in zones this epoch")
	}
	body, _ := json.Marshal(collapseIn{SpawnID: target.ID, Cell: target.Cell, Epoch: target.Epoch, Lat: target.Lat, Lon: target.Lon})
	post := func(dev string) (int, spawn.CollapseResult, map[string]any) {
		req, _ := http.NewRequest("POST", ts.URL+"/v0/collapse", bytes.NewReader(body))
		if dev != "" {
			req.Header.Set("X-Strata-Device", dev)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		var res spawn.CollapseResult
		var errBody map[string]any
		if resp.StatusCode == 200 {
			_ = json.NewDecoder(resp.Body).Decode(&res)
		} else {
			_ = json.NewDecoder(resp.Body).Decode(&errBody)
		}
		return resp.StatusCode, res, errBody
	}
	if st, _, _ := post(""); st != 400 {
		t.Fatalf("no device header: %d", st)
	}
	st, res, _ := post("device-alpha")
	if st != 200 || res.Item.Name == "" {
		t.Fatalf("collapse: %d %+v", st, res)
	}
	if st, _, e := post("device-alpha"); st != 409 {
		t.Fatalf("second claim: %d %v", st, e)
	}
	if st, _, _ := post("device-beta-2"); st != 200 {
		t.Fatalf("another device: %d", st)
	}
}

func ftoa(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}
