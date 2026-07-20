# shackboard

A web-based Shackboard (ham shack dashboard) workalike: world map with
day/night grayline, UTC + local clocks, a per-band propagation-conditions
summary, and live DX cluster spots with click-through QRZ callsign lookup.

Single Go binary, stdlib `net/http` only, no database. The frontend
(`web/`) is embedded into the binary via `go:embed`.

## Run locally

```
SHACKBOARD_CLUSTER_CALL=YOURCALL go run .
```

Then open http://localhost:8093/.

## Config

| Var | Default | Purpose |
|---|---|---|
| `SHACKBOARD_PORT` | `8093` | listen port |
| `SHACKBOARD_CLUSTER_HOST` | `dxc.ve7cc.net:23` | DX cluster telnet endpoint |
| `SHACKBOARD_CLUSTER_CALL` | *(required)* | callsign used to log into the cluster |
| `SHACKBOARD_SPACEWX_URL` | `https://www.hamqsl.com/solarxml.php` | propagation feed |
| `SHACKBOARD_SPOT_BUFFER_SIZE` | `200` | spot ring buffer capacity |
| `SHACKBOARD_SPOT_MAX_AGE` | `2h` | spots older than this are dropped |
| `SHACKBOARD_DEBUG` | unset | log unmatched cluster lines |

Your home QTH isn't backend config — enter your callsign in the UI once
(saved to localStorage); it's resolved via the sibling `qrzlook` service at
`/qrz/lookup/{callsign}`.

## Deploy

```
./deploy.sh peter@fimblefowl.co.uk
```

Builds for linux/amd64, copies the binary over, restarts the
`systemctl --user` service. First-time setup on the server:

```
mkdir -p ~/.config/systemd/user
cp shackboard.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now shackboard
```

Then add the location blocks in `nginx.conf.snippet` to
`/etc/nginx/sites-enabled/fimblefowl.conf` (both `/shackboard/` and `/qrz/`
— the latter isn't exposed on this domain yet even though the qrzlook
service itself is already running), and
`sudo nginx -t && sudo systemctl reload nginx`.
