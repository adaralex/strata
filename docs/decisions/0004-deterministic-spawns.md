# 0004 · Deterministic spawns, conditions and the drop

**Status:** accepted · **Date:** 2026-09-20 · **Affects:** PLAN.md §4, §5, §7, §10, §11;
CLAUDE.md phase 0 track 3

## Decision

Spawns are a pure function `spawns(cell_r8, epoch, digest)` in `server/spawn`, seeded by a
keyed hash of the three arguments and a server secret. No spawn row is ever stored. The
only state is one claim per device per spawn and the last fix per device for the speed
gate, in memory for phase 0 (`spawn.MemoryState`).

## Choices PLAN.md left open

- **Hash.** PLAN §4 says siphash. HMAC-SHA-256 truncated to 64 bits is in the standard
  library and runs once per cell; a splitmix64 stream seeded from it drives every draw,
  and it is identical on every platform. Slot `i` is fully determined by `(seed, i)`.
- **Conditions, phase 0 subset** (`server/cond`). Solar altitude from the NOAA
  low-precision formula, moon illumination from the mean synodic cycle, mean solar time
  for the after-midnight flag, propagation piecewise on solar phase with a moon bonus.
  Weather fields exist and are pinned to fair (PLAN §5 fallback). The digest packs the
  regional buckets only; per-cell statics are predicates, not digest.
- **Count** is `base + floor(log2(1 + poi_count) / 2)`, 3 in a village to 8 in a city
  centre, capped at 12, ×1.5 in wild cells, halved after local midnight in urban band 0
  to 1 (PLAN §11). A `value_mul` rises as density falls (PLAN §10), and shifts tiers up.
- **Civilization** is a weighted pick over the derived top four under the epoch's
  propagation. The folded residual is dropped from the pick, since we cannot know which
  eleven it hides. Track 2 may revisit that.
- **Authenticity** is per spawn, not per cell. Confluence when the top three weights sit
  within 0.1 of each other. Otherwise the chance of a Drift hybrid is 0 at purity ≥ 0.5,
  1 at purity ≤ 0.35, linear between, so night raises hybrids from a minority to a
  majority at Cugnaux without making them universal. The hybrid partner is the strongest
  other civilization. Accessioned and Sited are decided at collapse from the player's
  fix. The first cut made authenticity a per-cell band, and the whole map flipped to
  Drift at night; that is why it is a roll.
- **Placement** picks one of the cell's 49 r10 children that is not in the exclusion
  raster, then a point within 30 m of its centre. Private homes are still a follow-up:
  the snapshot carries no building footprints yet.
- **Collapse** re-derives the spawn for the claimed epoch (current or the one before),
  requires the fix within 40 m of the spawn and outside any exclusion geometry (the
  exact test of 0003), applies the speed gate from the device's last two fixes
  (non-negotiable 3, a first cut), records the claim, and derives the item from
  `hash(secret, spawn, device)`. A retried request yields the same item.
- **The drop** follows PLAN §7: civilization × archetype × tier × affixes × provenance.
  Inside a beacon with a curated vector, the civilization is drawn from the beacon's
  holdings and the item is Accessioned, or Sited at an archaeological site, stamped with
  the institution. Hybrids get the dominant archetype, an epithet from a written table
  per ordered pair, and no fact panel at all (non-negotiable 5). Edible-tagged monsters
  drop consumables half the time.
- **Content is data.** `rules/spawn.json` (every number above), `rules/bestiary.json`
  (monsters with `when` predicates in PLAN §5's rule shape), `rules/items.json`
  (archetypes with separate `fact` and `fiction`, hybrid epithets, edibles). The phase 0
  bestiary blends each civilization's material with the folklore of the ground it
  surfaces on: the gold of Tolosa and the Tectosages, the Bécut, Tantugou, the matagot,
  the hadas, Jean de l'Ours, the drac of the Garonne, the Cocagne pastel trade, the
  norias of Cugnaux. Each entry's `note` names its source; that text is not the fiction.

## The service

`server/cmd/worldd` serves lookups, spawns and collapses over HTTP for the walk test, with
an anonymous `X-Strata-Device` header that is not identity (non-negotiable 1; accounts
arrive in phase 1). `STRATA_SECRET` must be the same on every server answering for one
world, or players see different monsters.

## Deferred

Weather, the rule digest memoisation (rules are few enough to evaluate per call), private
homes at placement, the speed gate from activity recognition, and persistence of claims.
