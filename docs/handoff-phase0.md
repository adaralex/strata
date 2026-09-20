# Phase 0 hand-off · 2026-09-20

For the local Claude Code session with Unity access. Read `CLAUDE.md` first; this note
says where phase 0 stands and what to do next, in order. Everything here is on branch
`claude/focused-ptolemy-ettdfn`.

## State

| Track | State | Proof |
| --- | --- | --- |
| 1 World build | Done on the full Midi-Pyrénées extract | `worldq lookup 43.5365 1.3444` resolves in under 100 µs with fifteen civilizations |
| 2 Drift solve | Done on the planet at r5 | `drift verify`: 2,016,842 cells, zero unreached |
| 3 Deterministic spawns | Done; `worldd` serves lookup, spawns, collapse | tests in `server/spawn`, `server/cmd/worldd` |
| 4 Walk test | Source written, first phone run done: 488 m, two cells, zero fights | `client/Assets/Strata`, fixes for the two faults are committed but not yet built |

The first phone run found two faults, both fixed in the last commit and not yet verified
on a device:

1. Spawns were placed anywhere in a cell, mostly in gardens; they now stand within 5 m
   of a pedestrian way. Snapshot format is 4, so the snapshot must be rebuilt.
2. Taps read `UnityEngine.Input`, which throws under the Input System package and killed
   `Update`. Taps now go through `TapCatcher` and the UI event system, and the status line
   shows the nearest spawn on every fix.

## Next steps, in order

1. **Server side, once.**
   ```sh
   git pull
   go run ./worldbuild/cmd/worldbuild -osm midi-pyrenees-latest.osm.pbf -out out/midi-pyrenees -drift out/drift/drift.bin
   export STRATA_SECRET='the-same-string-every-time'
   go run ./server/cmd/worldd -snapshot out/midi-pyrenees -rules rules -addr :8080
   ```
   Watch the `walkable:` line in the build log; expect low millions of r10 cells. Keep
   `worldd` running for the editor and the phone (Tailscale IP for the phone).

2. **In Unity, editor first.** Pull into the project's `Assets/Strata`. Recompile. Fix
   anything the console reports; the likely places are `Map/MapAdapter.cs` (GO Map API
   names) and TextMeshPro's `enableWordWrapping` (obsolete, a warning). Press Play with GO
   Map's location simulation at 43.5365, 1.3444 and the Server URL pointing at `worldd`.
   Check, in this order: the ground strip shows Hallstatt / Phoenicia / Etruria / Scythia;
   markers appear on streets, not in blocks; the status line reads `nearest: …, N m`;
   clicking a marker within 40 m opens the fight; Auto-resolve posts the collapse and the
   item card shows a fact panel in Spectral above a fiction panel in the sans; End walk
   shows the summary and saves a JSON file.

3. **Device build.** IL2CPP, ARM64, minSdk 26, portrait, cleartext allowed, fine location.
   Install, set the Server URL to the Tailscale address, confirm `connected to …` on
   mobile data with Wi-Fi off.

4. **Shakedown walk, you.** Repeat the 488 m. The pass condition is one fight and one item
   card on the phone, and a saved debrief with the walk log. If a tap still does nothing,
   read the status line: it now always says something.

5. **The real walk, someone not on the team.** The loop in decision record 0003: Hôtel de
   ville, Noria du Parc du Manoir, Château de Maurens, Le Majorat, Médiathèque Clémence
   Isaure, back through the Vivier parks, about 3.5 km. Ask them to tap what they like
   and to answer the three questions honestly. The JSON they share is the phase 0 gate.

## Things that will need judgement, not fixing

- **Hybrids at Cugnaux.** Purity is 0.38 by day and 0.29 at night, so about a fifth of
  spawns are hybrids by day and three quarters at night. If that feels like too much
  invention on Gaulish soil, the lever is the isthmus corridor cost in
  `data/cores/phoenicia.geojson`, not the authenticity thresholds.
- **Elite and boss frequency.** 60 % of cells hold an elite each epoch; bosses roughly one
  in four epochs per neighbourhood. `rules/spawn.json`, `ranks`.
- **The residual.** Eleven civilizations are folded into one number per cell and cannot
  spawn. A Kemet spawn in Cugnaux at 2 a.m. is either the point of night play or noise;
  decision record 0004 leaves it open.
- **Content facts.** Every `note` in `rules/bestiary.json`, every `fact` in
  `rules/items.json` and the curated holdings in `data/beacons/toulouse.json` were written
  from memory and need a researcher's pass before anyone outside the team reads them.

## What not to do

Do not add accounts, payments, museum partnerships, combat depth, art or the event
framework (CLAUDE.md). Do not let the client decide a range, a drop, an exclusion or the
speed gate; it displays what `worldd` returns. Do not commit the Unity project folder or
the GO Map asset; `client/Assets/Strata` and `client/Packages/manifest.json` are the
repo's part.
