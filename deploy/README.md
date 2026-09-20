# Deploying worldd for a walk

The walk test needs `worldd` reachable from the phone over mobile data. Three options,
cheapest first. In every case set `STRATA_SECRET` to the same value on every start, or
the spawns change under the tester when the server restarts.

## Tailscale from the laptop at home

Install Tailscale on the laptop and the phone, same tailnet. Start `worldd` as usual and
keep the laptop awake. In the app, Server, enter `http://<laptop tailscale ip>:8080`
(`tailscale ip -4`). Private, encrypted, no ports opened.

## A small VPS with Docker

```sh
# on your machine: the snapshot is data, copy it up (check its size first: du -sh out/midi-pyrenees)
rsync -av out/midi-pyrenees/ vps:/srv/strata/snapshot/

# on the VPS
git clone <repo> strata && cd strata
docker build -t strata-worldd .
docker run -d --name worldd --restart unless-stopped \
  -p 8080:8080 \
  -e STRATA_SECRET='a-long-random-string' \
  -v /srv/strata/snapshot:/strata/snapshot:ro \
  strata-worldd
curl -s localhost:8080/healthz
```

Open port 8080 in the VPS firewall, or put Caddy in front for https on 443:

```
walk.example.org {
    reverse_proxy localhost:8080
}
```

Claims and the speed gate live in memory (phase 0); a restart forgets them.

## Cloudflare quick tunnel

`cloudflared tunnel --url http://localhost:8080` prints a public https URL. No account,
no setup, but public and unauthenticated, and the URL changes every run. For one
afternoon only; take it down afterwards.
