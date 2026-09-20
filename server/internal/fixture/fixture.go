// Package fixture builds the Cugnaux test snapshot for server tests, the
// same way worldbuild does from worldbuild/testdata/cugnaux.osm.
package fixture

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/worldbuild/cells"
	"github.com/adaralex/strata/worldbuild/classify"
)

// Root is the repository root, found from this file's location.
func Root() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

// Snapshot builds the fixture into a temp dir and returns the dir.
func Snapshot(t *testing.T) string {
	t.Helper()
	root := Root()
	civsJSON, err := os.ReadFile(filepath.Join(root, "rules", "civs.json"))
	if err != nil {
		t.Fatal(err)
	}
	civs, err := world.ParseCivs(civsJSON)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := classify.LoadRules(filepath.Join(root, "rules", "poi_classes.json"))
	if err != nil {
		t.Fatal(err)
	}
	layers, err := cells.LoadLayers(filepath.Join(root, "data", "cores"), civs)
	if err != nil {
		t.Fatal(err)
	}
	curated, err := cells.LoadCurated(filepath.Join(root, "data", "beacons"), civs)
	if err != nil {
		t.Fatal(err)
	}
	civTags, err := cells.LoadCivTags(filepath.Join(root, "data", "civ_tags.json"), civs)
	if err != nil {
		t.Fatal(err)
	}
	lists, err := classify.LoadLists(filepath.Join(root, "data"), rules)
	if err != nil {
		t.Fatal(err)
	}
	ex, err := classify.ReadOSM(context.Background(), filepath.Join(root, "worldbuild", "testdata", "cugnaux.osm"), rules, nil)
	if err != nil {
		t.Fatal(err)
	}
	res := classify.Run(ex, rules, lists)
	snap, _, err := cells.Build(cells.Inputs{BuildID: "fixture", Civs: civs, Rules: rules, Layers: layers, Curated: curated, CivTags: civTags, Classified: res, Bounds: ex.Bounds})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := snap.Write(dir, civsJSON); err != nil {
		t.Fatal(err)
	}
	return dir
}

// World builds the fixture and loads it.
func World(t *testing.T) *world.World {
	t.Helper()
	w, err := world.Load(Snapshot(t))
	if err != nil {
		t.Fatal(err)
	}
	return w
}
