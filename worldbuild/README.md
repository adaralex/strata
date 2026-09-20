# worldbuild

Phase 0 track 1: one OSM extract in, one H3 r8 snapshot out (PLAN.md §14 steps 1 to 4).

## Run the Midi-Pyrénées build

```sh
# 1. The extract, unclipped (decision record 0003). A few hundred MB.
curl -L -o midi-pyrenees-latest.osm.pbf \
  https://download.geofabrik.de/europe/france/midi-pyrenees-latest.osm.pbf

# 2. Build. Three streaming passes over the file, then the cell build.
go run ./worldbuild/cmd/worldbuild -osm midi-pyrenees-latest.osm.pbf -out out/midi-pyrenees

# 3. Query. Cugnaux town hall, by day and at full propagation.
go run ./server/cmd/worldq -snapshot out/midi-pyrenees lookup 43.5365 1.3444
go run ./server/cmd/worldq -snapshot out/midi-pyrenees lookup 43.5365 1.3444 -p 1

# What is within a 3 km walk, beacons first: the second-zone question.
go run ./server/cmd/worldq -snapshot out/midi-pyrenees nearby 43.5365 1.3444 -radius 3000

# Acceptance: lookups under 5 ms.
go run ./server/cmd/worldq -snapshot out/midi-pyrenees bench
```

`-bbox minLat,minLon,maxLat,maxLon` overrides the cell coverage (the PBF header bbox by
default). `-rules` and `-data` point at the repo's `rules/` and `data/` directories.

## What it reads

| Input | Purpose |
| --- | --- |
| `rules/civs.json` | The fifteen, with `lambda_km` and `k` (PLAN §4, §5) |
| `rules/poi_classes.json` | POI classes, exclusions, terrain selectors (PLAN §9, §11) |
| `data/cores/*.geojson` | Core and periphery polygons, corridor lines, per civilization |
| `data/beacons/*.json` | Curated museum holdings, matched by ref or name |
| `data/civ_tags.json` | `historic:civilization=*` to civilization key |
| `data/blocklist.geojson`, `data/blocklist_names.json` | Sites of conscience and name substrings that suppress beacons |
| `data/worship_optin.json`, `data/memorial_whitelist.json` | Refs allowed despite matching an exclusion |

## What it writes

A snapshot directory: `cells.bin` (header, sorted r8 keys, 32-byte records, beacon
adjacency, sorted r10 exclusion set), `beacons.json`, `pois.json`, and a copy of
`civs.json` so the snapshot is self-contained. The record layout is documented in
`server/world/record.go`.

## Pipeline

1. **Read** (`classify.ReadOSM`): relations, then ways, then nodes, keeping only features
   whose tags match a class, exclusion or terrain selector, and only the node
   coordinates those features need. Multipolygon outer rings are chained; broken rings
   are dropped rather than guessed at.
2. **Classify** (`classify.Run`): exclusions first, then the highest-priority class plus
   any stackable one (a museum with a library is beacon and scriptorium). Memorials are
   beacons only if whitelisted; beacons with a blocklisted name are suppressed.
3. **Build** (`cells.Build`): per r8 cell, the cost to each civilization from the nearest
   layer, ranked by cost over base lambda, top four kept and the tail folded into a
   residual. Exclusion geometries rasterised to r10. POIs suppressed by exact geometry.
   Density over the k-ring 1, service mask, beacon adjacency and grade, water proximity,
   wild and coast flags.

## Tests

`go test ./...` runs against `worldbuild/testdata/cugnaux.osm`, a hand-written fixture
shaped like the town centre, so no download is needed. `BenchmarkLookup` in
`server/world` is the latency acceptance check.
