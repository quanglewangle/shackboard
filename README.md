# shackboard

A web-based Shackboard (ham shack dashboard) workalike: world map with
day/night grayline, UTC + local clocks, a per-band propagation-conditions
summary, live DX cluster spots and POTA/SOTA activator spots (both with
click-through QRZ callsign lookup), and "worked before" highlighting
synced automatically from your QRZ Logbook.

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
| `SHACKBOARD_CLUSTER_HOST` | `dxspider.co.uk:7300` | DX cluster telnet endpoint |
| `SHACKBOARD_CLUSTER_CALL` | *(required)* | callsign used to log into the cluster |
| `SHACKBOARD_SPACEWX_URL` | `https://www.hamqsl.com/solarxml.php` | propagation feed |
| `SHACKBOARD_SPOT_BUFFER_SIZE` | `200` | spot ring buffer capacity |
| `SHACKBOARD_SPOT_MAX_AGE` | `2h` | spots older than this are dropped |
| `SHACKBOARD_DEBUG` | unset | log unmatched cluster lines |
| `SHACKBOARD_POTA_URL` | `https://api.pota.app/spot/activator` | POTA activator-spot feed |
| `SHACKBOARD_SOTA_URL` | `https://api2.sota.org.uk/api/spots/100/all` | SOTA spot feed |
| `SHACKBOARD_QRZ_LOGBOOK_KEY` | unset | QRZ Logbook Data API key — enables "worked before" sync; disabled entirely if unset |
| `SHACKBOARD_QRZ_LOGBOOK_URL` | `https://logbook.qrz.com/api` | QRZ Logbook API endpoint |

Your home QTH isn't backend config — enter your callsign in the UI once
(saved to localStorage); it's resolved via the sibling `qrzlook` service at
`/qrz/lookup/{callsign}`.

## API

| Method | Path | Notes |
|---|---|---|
| GET | `/api/spacewx` | per-band propagation conditions |
| GET | `/api/spots` | DX cluster spots, `?limit=N` (default 100) |
| GET | `/api/park-spots` | merged POTA + SOTA spots |
| GET | `/api/log` | worked-before index status: `{"loaded","qso_count","synced_at"}` |
| GET | `/api/log/contacts/{call}` | past QSOs with `call`, most recent first: `{"call","loaded","contacts":[{"date","time","band","mode"}]}` |
| GET | `/health` | liveness + subsystem status |

`/api/spots` and `/api/park-spots` entries are decorated with `worked_any`/
`worked_band` booleans against the log currently synced from QRZ.

The worked-before index is populated automatically — no manual step. If
`SHACKBOARD_QRZ_LOGBOOK_KEY` is set, shackboard fetches your whole QRZ
Logbook via QRZ's Logbook Data API (get a key from your QRZ account >
Logbook > Settings > API — different from the XML lookup key qrzlook
uses) once at startup and hourly thereafter.

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
