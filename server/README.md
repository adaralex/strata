# server

Go services (PLAN.md §13). Phase 0 has one: `worldd`.

## worldd

```sh
export STRATA_SECRET='a-long-random-string-shared-by-every-server'
go run ./server/cmd/worldd -snapshot out/midi-pyrenees -rules rules -addr :8080
```

Without `STRATA_SECRET` it starts with a random secret and warns: spawns then change on
restart and differ from `worldq spawns`.

| Method and path | Purpose |
| --- | --- |
| `GET /healthz` | Liveness and the snapshot build id |
| `GET /v0/lookup?lat=&lon=[&p=]` | Weights, purity, services, beacons in range, exclusion, conditions. `p` overrides propagation |
| `GET /v0/spawns?lat=&lon=` | Conditions and the spawns of the cell and its six neighbours, nearest first, with `distance_m`. Send `X-Strata-Device` so the speed gate sees your fixes; `speed_gated` says whether interaction is disabled |
| `POST /v0/collapse` | Body `{"spawn_id","cell","epoch","lat","lon"}`, header `X-Strata-Device`. Returns the spawn, the item and the beacon if any |

Collapse statuses: 400 bad cell or future epoch, 403 too far, excluded or speed-gated,
404 unknown spawn, 409 already collapsed by this device, 410 epoch expired.

```sh
curl -s 'localhost:8080/v0/spawns?lat=43.5365&lon=1.3444' -H 'X-Strata-Device: my-phone-0001' | jq '.spawns[0]'
curl -s localhost:8080/v0/collapse -H 'X-Strata-Device: my-phone-0001' \
  -d '{"spawn_id":"...","cell":"8839601945fffff","epoch":1988784,"lat":43.5365,"lon":1.3444}' | jq .item
```

## worldq

```sh
go run ./server/cmd/worldq -snapshot out/midi-pyrenees lookup 43.5365 1.3444 [-p 0.5]
go run ./server/cmd/worldq -snapshot out/midi-pyrenees nearby 43.5365 1.3444 [-radius 3000]
go run ./server/cmd/worldq -snapshot out/midi-pyrenees spawns 43.5365 1.3444 [-t 2026-09-20T22:30:00Z]
go run ./server/cmd/worldq -snapshot out/midi-pyrenees bench
```

`spawns` uses `STRATA_SECRET` when set, else a fixed development secret.

## Packages

- `world`: r8 cell record, weights, snapshot codec, lookup. `world/h3x` is the only place
  with spatial maths.
- `cond`: epoch, solar phase, moon, propagation, digest (PLAN §5, phase 0 subset).
- `spawn`: deterministic spawns, rule predicates, placement, collapse and the drop
  (decision record 0004). Content in `rules/spawn.json`, `bestiary.json`, `items.json`.
