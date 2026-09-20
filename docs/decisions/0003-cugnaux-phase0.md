# 0003 · Cugnaux is the phase 0 city, built from the Midi-Pyrénées extract

**Status:** accepted · **Date:** 2026-09-20 · **Affects:** PLAN.md §19 (phase 0), §21 open
question 2, CLAUDE.md open questions

## Decision

Phase 0 builds the world for Cugnaux (Haute-Garonne, south-west of Toulouse) from the
whole Geofabrik `europe/france/midi-pyrenees` extract, unclipped. The three hand-drawn
civilization layers are:

- **Hallstatt** as the soil. Cugnaux sits in the periphery ring (Gaul, the La Tène
  world; the Vieille-Toulouse oppidum and the Fenouillet torcs are the local anchors),
  not the Alpine and Bohemian core.
- **Phoenicia** as a corridor: the Mediterranean littoral and the Aude to Garonne isthmus
  (Narbonne, Carcassonne, Toulouse, Bordeaux), the overland route by which Mediterranean
  goods reached the Toulouse basin.
- **Etruria** as the distant core that shows the drift gradient and what raising
  propagation does.

Kemet arrives in Toulouse as a museum beacon (Musée Georges-Labit), not as soil.

## The record and the schema

The POI classification schema is `rules/poi_classes.json` and the cell record is
documented in `server/world/record.go`. Choices made here that PLAN.md left open:

- **Costs are stored, weights are derived.** PLAN §4 describes storing four weights and a
  purity byte; §5 then says costs must be static and only lambda moves. The record
  stores four u16 costs plus a folded residual, and weights and purity are five
  exponentials at query time. Track 1 fills costs with great-circle distance to the
  layer; track 2 replaces that with the terrain Dijkstra and nothing downstream changes.
- **Top four ranked by cost over base lambda**, not raw cost, so a maritime civilization
  with a long reach is not dropped by a nearer landlocked one.
- **Exclusions live at r10** (66 m cells) as a sorted set; the r8 record only counts how
  many of its 49 children are excluded. POIs are suppressed by exact geometry (inside
  the feature plus its buffer), with the r10 set as the spatial index, because a war
  memorial and a church sit on the same square as the bakery in every French town.
- **Private homes are placement-only.** Baking `building=house` into the r10 set blanks
  every residential street. Spawn placement (track 3) checks building polygons instead.
- **Beacons carry their own civilization vector**, from a curated table, then the OSM
  `historic:civilization` tag, then the soil. Unknown sites still shift tier and rarity.
- **Hearth is one class.** Food shops and restaurants heal, drop edibles, and bias spawns
  toward edible-loot monsters. Inns are a dwell effect; pharmacies grant potions and
  cures; workshops and repair shops are the forge.

## Amended after the first Midi-Pyrénées build (2026-09-20)

- **Zero-buffer exclusions use centre containment.** Places of worship (buffer 0) claim
  only the r10 cells whose centre lies inside the footprint, one cell minimum. Overlap
  containment swallowed the square in front of the church at the Cugnaux town hall.
  Buffered exclusions (schools, hospitals) keep overlap plus their disk.
- **The snapshot names the zone** that excluded each r10 cell, so `worldq` can say why a
  point is dead.
- **Garages are not forges.** `shop=car_repair` and `motorcycle_repair` are dropped; the
  forge is hardware, workshop crafts and bicycle repair. Fifteen garages within 3 km of
  the town hall were drowning two hardware stores.
- **Private land is never a service or a spawn zone.** A global `skip` rule drops any
  feature with `access=private|no` from every class, and `leisure=garden` counts as wild
  only with `garden:type` community, botanical or public. Nine unnamed private gardens
  were wild zones. Exclusions and terrain still read private features.

## Open

- ~~The second walk-test zone.~~ Answered by the first build: five beacons within 3 km
  of the town hall, all inheriting soil. The walk-test loop is Hôtel de ville, Noria du
  Parc du Manoir (grade 2, 190 m), Château de Maurens (grade 1, 730 m), Le Majorat
  (grade 3, 1.35 km), Médiathèque Clémence Isaure (1.5 km), back through the Vivier
  parks: about 3.5 km, soil zone then beacon halos.
- **Curated holdings are unverified.** `data/beacons/toulouse.json` is marked `verify`.
