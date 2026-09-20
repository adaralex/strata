# 0002 · The Android client is Unity with GO Map

**Status:** accepted · **Date:** 2026-09-20 · **Supersedes:** PLAN.md §12 (Kotlin, Compose,
MapLibre, 2D), the "Client" and "Map" rows of the CLAUDE.md locked decisions

## Decision

The client is built in Unity, using the "GO Map - 3D Map for AR Gaming" asset for the
world map and visuals. The Kotlin/Compose/MapLibre stack and the "2D, no Unity" lock are
withdrawn.

## What does not change

- Android remains the only launch platform. Unity makes an iOS build cheap later, but
  it is still a later client, not a second implementation.
- Everything valuable is decided on the server (CLAUDE.md non-negotiable 1). The client
  choice has no effect on the world build, the cell record, spawns or loot.
- Content is data. Civilizations, spawn rules, items and lore live under `/data` and
  `/rules` and are read by the client, never compiled into it.

## Consequences to own

PLAN.md §12 argued for 2D on four points. Each now needs an answer during phase 0 or 1:

1. **Install size.** §12 budgets 45 MB against ~180 MB for an engine build. Track it as
   a metric from the first Unity build.
2. **Battery and passive collection.** §3 makes background collection along a commute
   the retention mechanic, and §12 wanted a plain foreground service rather than an
   engine loop fighting Doze. The passive collector should be a native Android service
   beside the Unity activity, not Unity code.
3. **Hybrid items.** §7 costs the 210 civilization pairs as composites of layered 2D art
   (silhouette, pattern, palette, prop). With 3D or GO Map visuals that costing is void
   and needs redoing before phase 1 art direction.
4. **The AR question.** §12 ruled out AR at launch. GO Map is an AR-oriented asset;
   whether AR is now in scope is a separate decision and is not made here.

The phase 0 walk test (track 4) will be a bare Unity scene: GO Map, location, placeholder
fights, real drops from the world service.
