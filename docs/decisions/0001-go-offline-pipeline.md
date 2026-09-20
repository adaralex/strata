# 0001 · The offline world build is Go

**Status:** accepted · **Date:** 2026-09-20 · **Affects:** PLAN.md §14, CLAUDE.md conventions

## Decision

The worldbuild pipeline (OSM extract to POI classification to H3 cell snapshot) is
written in Go, in the same module as the server. No Python in the repo.

## Why

- The pipeline and the world service share the H3 helper (`server/world/h3x`) and the
  snapshot codec (`server/world`). One implementation of the record layout, one set of
  tests, no drift between what the build writes and what the service reads.
- The drift solve (track 2) is fifteen Dijkstra runs over a planet-sized graph. It is
  compute-bound and wants goroutines, not a notebook.
- The phase 0 prototype is throwaway but is written in the language the real thing will
  be in (CLAUDE.md conventions), so the parts that survive can survive.

## Consequences

- OSM reading uses `paulmach/osm` (PBF and XML), geometry uses `paulmach/orb`, H3 uses
  `uber/h3-go` v4, which is a CGO binding and needs a C compiler on build machines.
- The fixture for tests is an OSM XML file (`worldbuild/testdata/cugnaux.osm`), so tests
  do not need a Geofabrik download.
