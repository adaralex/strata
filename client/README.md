# client

Phase 0 track 4: the walk test (PLAN.md §19, decision record 0002). Unity 6, GO Map, one
scene, no art. The repo holds the source under `Assets/Strata` and a package manifest; the
Unity project itself is created locally, because GO Map is a paid asset and Unity's
generated project files do not belong in git until a build has succeeded.

## Set up, once

1. **Create the project.** Unity Hub, Unity 6, the 3D (Built-in) template, Android build
   support installed. Any folder outside this repo.
2. **Copy the source.** Copy `client/Assets/Strata` into the project's `Assets/`. Merge
   `client/Packages/manifest.json` into the project's: the line that matters is
   `com.unity.nuget.newtonsoft-json`.
3. **Import GO Map** from the Package Manager, My Assets. Open one of its demo scenes once
   so its prefabs and tile settings are created.
4. **Define `STRATA_GOMAP`.** Project Settings, Player, Android, Scripting Define Symbols.
   Without it the code compiles against a flat plane, which is useful for a first run in
   the editor but is not the map.
5. **Android settings.** IL2CPP, ARM64, minimum API 26, portrait, Internet access
   Required, and in the Android manifest or Player settings the fine location permission.
   Add `android:usesCleartextTraffic="true"` if `worldd` is on plain http on a hotspot.

## The scene

Create `Assets/Strata/Scenes/Walk.unity` with three things:

- **A camera** tagged MainCamera. GO Map's demo scenes come with a camera rig that
  follows the player; reuse it.
- **The GO Map object** from the GO Map prefab, tile source set to an OSM style, its
  LocationManager set to follow the device. For the editor, GO Map's location simulation
  works: set its demo coordinates to Cugnaux, 43.5365, 1.3444.
- **An empty GameObject `Strata`** with `MapAdapter` and `WalkController`. Assign the GOMap
  component to `MapAdapter.goMap`, the camera to `WalkController.worldCamera`, and
  optionally a `StrataSettings` asset (Assets, Create, Strata, Settings) with the
  `worldd` URL. The URL is also editable in the app under **Server**.

Everything else, the ground strip, the markers, the fight, the card, the debrief, is
built at runtime by `WalkController`. If the scene runs but shows a flat grey plane,
`STRATA_GOMAP` is not defined or `MapAdapter.goMap` is empty.

If your GO Map version's API differs, `Scripts/Map/MapAdapter.cs` is the only file that
names it: two calls, `Coordinates.convertCoordinateToVector3()` and the location
manager's current location.

## Run

```sh
# on the laptop, same hotspot as the phone
export STRATA_SECRET='a-long-random-string'
go run ./server/cmd/worldd -snapshot out/midi-pyrenees -rules rules -addr :8080
```

Set the app's server URL to `http://<laptop-ip>:8080`, walk. **End walk** shows the walk
log and the three debrief questions and saves a JSON file to the app's persistent data
path, offered through the share sheet. That file is what phase 0 is judged on.

## After the first phone walk

Two fixes landed after the first 488 m: taps go through the UI event system
(`TapCatcher`) because reading `UnityEngine.Input` throws when a Unity 6 project has the
Input System package active, and the status line now shows the nearest spawn and its
distance on every fix. Server side, spawns now stand on streets and paths; rebuild the
snapshot (`worldbuild`, snapshot format 4) and restart `worldd` before the next walk.

## What the client never does

It never decides a drop, a range, an exclusion or the speed gate; it displays what
`worldd` returns and posts the fix with a collapse. Fact and fiction on an item card are
separate panels in separate typefaces (Spectral for facts, the default sans for the
Anvil's lines); a hybrid shows no fact panel. Auto-resolve is always on the fight screen.

## Working with a local Claude Code session and a Unity MCP

The editor work (scene wiring, Play mode, console) is best done from a Claude Code
session running on the same machine as Unity, with a Unity MCP bridge configured over
stdio. This README is the checklist that session should follow; `MapAdapter.cs` is where
GO Map API differences will surface first.
