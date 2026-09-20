# CLAUDE.md — STRATA

Project context for Claude Code. Read this first, then `PLAN.md` for anything it points at.

## What this is

A GPS role-playing game for Android. The player's real location resolves to a weighted
mixture of fifteen ancient civilizations; monsters and loot come from that mixture. Real
city services read from OpenStreetMap drive the loop (bakeries heal, banks store, clothes
shops re-equip). Museums act as loot beacons for the civilizations they actually hold.

`PLAN.md` is the full design and build plan, 21 sections. It is the source of truth for
intent. It is **not** a spec — where it is vague, ask rather than invent.

## Status

Pre-production. Phase 0 (see PLAN.md §19) is a **throwaway prototype** whose only job is to
answer one question: does walking a real route through real civilization zones feel like
anything? Nothing written in phase 0 is expected to survive. Do not build for reuse yet.

## Locked decisions

Do not relitigate these without being asked:

| Decision | Value |
| --- | --- |
| Platform | Android only for now; iOS is a later client, never a second implementation |
| Client | Unity, with the "GO Map - 3D Map for AR Gaming" asset for map and visuals (decision record 0002, supersedes PLAN.md §12). AR at launch is undecided |
| Map | GO Map inside Unity; the server never sees the map stack |
| Combat renderer | Unity scene; the Rive vs Compose spike is withdrawn |
| Server | Go, stateless services, server-authoritative for everything valuable |
| Storage | PostgreSQL 16 + PostGIS, Redis for hot state, NATS for events |
| Spatial index | H3. r8 for cell weights, r9/r10 for spawn placement, r5 for the weather cache |
| Identity | Play Games Services v2, wrapped in our own account id from day one |
| Civilizations | Fifteen. Phase 1 ships Etruria, Kemet, Hopewell deep |

## Non-negotiables

These are not preferences. Flag it loudly if a task would violate one.

1. **The server decides anything valuable.** Drops, accessions, translations, medal
   unlocks, leaderboard submissions. The client never tells the server what it looted, and
   a client-supplied Play Games player id is never treated as identity.
2. **Exclusion zones ship before content does.** No gameplay at schools, hospitals, police,
   military, prisons, cemeteries, memorials of death or atrocity, or private homes. Places
   of worship are opt-in per site. See PLAN.md §11.
3. **Speed gate.** Above ~15 km/h sustained, interaction is disabled. Distance only accrues
   toward medals below the gate.
4. **No timed-and-placed incentives.** Nothing rewards reaching a specific spot by a
   deadline. Natural events (eclipses) are the one exception and carry their own rules.
5. **Fact and fiction are visually separate.** Real artefact information and invented lore
   never share a typeface or a panel. Hybrid items carry no non-fiction claim at all.
6. **Accessibility is in scope from the start.** Auto-resolve combat, every daily objective
   completable within a 200 m radius, a non-distance medal line of equal prestige.

## Repo layout (target)

```
/client          Unity source: GO Map world map, walk-test scene; project created locally
/server          Go services: world, combat, player, beacon, season, trust
/worldbuild      Offline pipeline: OSM extract -> POI classify -> H3 cell weights
/proto           Protobuf schemas shared by client and server
/data            Hand-authored GeoJSON: civilization cores, corridors, blocklist
/rules           Spawn rule JSON, hot-reloadable (PLAN.md §5)
/docs            PLAN.md and decision records (docs/decisions/NNNN-*.md)
```

Today (tracks 1, 2 and 3): `/server/world` (the r8 cell record, weight derivation, snapshot
codec, lookup; the H3 helper in `h3x`), `/server/cond` (epoch, solar phase, propagation,
digest), `/server/spawn` (deterministic spawns, rule predicates, collapse and the drop),
`/server/cmd/worldd` (HTTP world service) and `/server/cmd/worldq` (query CLI),
`/worldbuild/classify` (OSM reader and POI classifier), `/worldbuild/cells` (the cell
builder), `/worldbuild/cmd/worldbuild` (the pipeline CLI), `/worldbuild/drift` and
`/worldbuild/cmd/drift` (the planet cost field from Natural Earth terrain), `/rules`
(civs, POI classes, spawn parameters, bestiary, items, drift costs), `/data` (fifteen civilization layers, curated Toulouse
beacons, block and allow lists). Go module at the repo root. See `worldbuild/README.md`
to run the Midi-Pyrénées build and the drift solve, `server/README.md` to serve it, and
`docs/handoff-phase0.md` for where phase 0 stands and what to do next, and
`client/README.md` to set up the Unity walk test (`/client/Assets/Strata`: source only; the
Unity project and the GO Map asset live outside git). Track 4's source is written; the
first editor wiring and device build are pending. Decisions so far are in `docs/decisions`.

## Phase 0 spike — the only work in scope right now

Five weeks, throwaway, no art. Four independent tracks:

1. **World build.** Take one Geofabrik metro extract. Classify the POI tag set in PLAN.md
   §9. Hand-draw three civilization core polygons. Produce H3 r8 weight vectors and a
   `purity` scalar. Acceptance: query any lat/lon, get back a plausible weight vector in
   under 5 ms.
2. **Drift solve.** Multi-source cost-distance from cores over a land/coast/ocean graph,
   store `cost_i` per cell, derive weights from a runtime `lambda`. Acceptance: nowhere on
   the planet returns zero weight, and raising `lambda` visibly pulls distant civilizations
   in. See PLAN.md §4 and §5.
3. **Deterministic spawns.** `spawns(cell, epoch, condition_digest)` from a hash, no stored
   spawn rows. Acceptance: two devices in the same cell see the same monsters.
4. **Walk test.** The actual point. Bare Unity scene, GO Map, location, placeholder
   fights, real drops. Acceptance: walk a real 3 km route through two zones and have
   someone who is not on the team tell you whether it felt like anything.

Do not build: accounts, payments, museums, combat depth, art, the event framework.

## Conventions

- Go: standard layout, `golangci-lint`, no ORM, `pgx` with hand-written SQL.
- Unity: C#, one scene per screen, no gameplay rules in the client; the passive collector
  is a native Android foreground service beside the Unity activity.
- Everything spatial goes through one H3 helper module. No ad-hoc lat/lon maths.
- Content is data, never code: civilizations, spawn rules, item archetypes and lore all
  live in versioned files under `/data` and `/rules`, hot-reloadable.
- Commit messages reference the plan section they implement, e.g. `worldbuild: drift solve (§4)`.
- Write the throwaway prototype in the language it will eventually be in. The offline
  pipeline is Go too (decision record 0001): it shares the H3 helper and the snapshot codec
  with the server, and the drift solve is compute-bound. No Python in the repo.
- Decisions that change a plan section or a convention get a record in `docs/decisions`.
- CI (`.github/workflows/ci.yml`) runs gofmt, build, vet, golangci-lint (`.golangci.yml`),
  race tests, the lookup benchmark and a build of the test fixture. Unity builds and
  Semgrep are not wired up yet. Keep it green.

## Open questions — ask, do not decide

Listed in full in PLAN.md §21. The ones that affect code:

- ~~Which metro area is the phase 0 city.~~ Decided: Cugnaux, built from the whole
  `europe/france/midi-pyrenees` extract, unclipped (decision record 0003). The first
  walk-test zone is Hallstatt periphery soil; the second is the beacon halos the first
  build found: the Noria du Parc du Manoir, Château de Maurens and Le Majorat
  (decision record 0003).
- Team size, which sets whether phase 0 runs four tracks in parallel or one at a time.
- Whether AR is in scope at launch now that the client is Unity and GO Map (decision
  record 0002).
