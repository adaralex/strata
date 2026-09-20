# 0005 · The drift solve

**Status:** accepted · **Date:** 2026-09-20 · **Affects:** PLAN.md §2 (drift), §4 (drift field,
purity), §5 (lambda); CLAUDE.md phase 0 track 2

## Decision

`worldbuild/drift` solves PLAN §4's cost-distance field for the whole planet: fifteen
multi-source Dijkstra runs over an H3 cell graph whose edges are priced by terrain.
`worldbuild -drift` folds the result into the cell record as the cost floor, so the
record, the derivation and everything downstream are unchanged from track 1.

## Choices

- **Resolution r5 by default** (2.0 M cells, 9.9 km edge), r6 behind `-res`. PLAN §4 says
  r6; at r5 the planet solves in about fifteen seconds and the field is 75 MB, and the
  drift is a smooth field whose gradients run over hundreds of kilometres. Costs are
  interpolated to r8 by inverse-distance weighting over the containing cell and its ring.
- **Terrain from Natural Earth 1:10m**, public domain, fetched from the GitHub mirror of
  the Natural Earth vector repository because naturalearthdata.com is not reachable from
  every build environment. Five layers: land, glaciated areas, physical regions (mountain
  ranges and deserts), river centrelines, lakes. Classes: ocean, land, coast, river, lake,
  mountain, desert, ice.
- **A scanline raster, not H3 polyfill.** A million-vertex coastline defeats
  polygon-to-cells, so the layers are even-odd filled onto a 0.05° grid once and cell
  centres read it. It is the one piece of geometry outside `h3x`, kept inside the drift
  package.
- **Costs in data** (`rules/drift.json`): land 1, river 0.6, mountain and desert 3, ice 6
  km-equivalent per km; ocean 2 scaled by 0.5 for maritime civilizations and 4 for
  landlocked ones, then by each civilization's own `ocean_mul` in `civs.json` (Lapita
  0.75). Coast is half price for sailors. Sources start at their layer cost, so a core is
  exactly 0 and a periphery 150.
- **All fifteen have layers now.** Twelve coarse cores and peripheries were authored from
  PLAN §2's soil-zone column to the same standard as the first three. Australia has none
  by design (PLAN §2) and fills from Lapita, Meluhha and Shang by drift.
- **The smear term.** The first solve made the mid-Atlantic pure Lapita and Baffin Island
  pure Hopewell: with exp(-cost/lambda) over thousands of kilometres the nearest
  civilization takes everything, and PLAN §4 expects purity near 0.3 mid-ocean. The
  derivation now uses `lambda_eff = lambda_base * (1 + k * propagation) + smear * c_min`
  with `c_min` the cell's cheapest cost and `smear` 0.5 in `civs.json`. Near a core it is
  a rounding error; far from everyone it widens every lambda alike, which is PLAN §2's
  fiction of a signal that "arrives smeared together" made literal. The residual is
  folded with the same term so the tail share inverts exactly by day.
- **Lambda retune.** Lapita's lambda drops from 1500 to 700 and its reach moves into the
  ocean cost, which is anisotropic, where a long lambda was isotropic and pulled Lapita
  into Mongolia. Scythia 700 to 500.
- **Authenticity thresholds** move to grounded 0.42 and drift 0.24 so Cugnaux, at purity
  0.38 by day and 0.29 at night, rolls hybrids about a fifth of the time by day and about
  three quarters at night.

## Acceptance

`drift verify`: 2,016,842 cells, zero unreached (cell, civilization) pairs. Nowhere on the
planet returns zero weight. `drift query` at propagation 0 and 1:

| Point | Day | Night |
| --- | --- | --- |
| Cugnaux | Hallstatt .38, Phoenicia .36, Etruria .15, Scythia .07 | .29 / .28 / .19 / .13 |
| Rome | Etruria .44, Phoenicia .30, Hallstatt .14, Minoa .08 | .29 / .25 / .18 / .15 |
| Iqaluit | Hopewell .42, Phoenicia .20, Lapita .10, Minoa .10 | .35 / .21 / .14 / .12 |
| Ulaanbaatar | Scythia .49, Shang .23, Lapita .16 | .37 / .23 / .21 |
| Mid-Atlantic | Phoenicia .56, Minoa .21, Lapita .09, Aksum .09 | .45 / .21 / .13 / .12 |
| São Paulo | Chavín .55, Lapita .19, Phoenicia .07 | .41 / .23 / .11 |
| Perth | Lapita .94, Meluhha .03, Aksum .02 | .81 / .06 / .08 |

Raising propagation lowers purity and raises every distant share, at every point.

## Open

Every number above is a first tuning. Lapita at 0.16 in Ulaanbaatar (over land from the
Island Southeast Asia periphery) and Perth at 0.94 Lapita are the two to look at first.
The twelve new layers need the same cartographer's pass as the first three.
