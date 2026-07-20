# shackboard

A web-based HamClock (ham shack dashboard) workalike. Deliberately **not**
named "hamclock" — that's the name of a real, well-known, unrelated ham
radio project (Elwood Downey WB0OEW's HamClock); this one is called
shackboard specifically to avoid confusion with it. Don't rename it back.

Single Go binary, stdlib `net/http` only, no database, no third-party Go
dependencies. Frontend is plain HTML/CSS/JS embedded via `go:embed`. See
`README.md` for the config/API reference — this file is about how the
codebase is put together and what to watch out for when extending it.

## Architecture

Every feature follows the same shape: a small `internal/<name>` package
with a `Poll(ctx, cache, client, ...)` goroutine that fetches an external
feed on a timer and caches the result behind a mutex, wired up in
`main.go`. Look at `internal/spacewx` first — it's the simplest example of
the pattern (poll an HTTP feed hourly, cache with `sync.RWMutex`, serve
stale-but-valid data on fetch error rather than a hard failure).

- `internal/cluster` — persistent telnet connection to a DX cluster node,
  reconnect/backoff, ring buffer of spots. `BandForFreqKHz` here is the
  one shared band-lookup table (`internal/adif` and `internal/parkspots`
  both import `cluster` just for this function — a minor layering smell,
  accepted rather than splitting out a fourth tiny package for one table).
- `internal/spacewx` — hourly poll of a propagation-conditions feed.
- `internal/parkspots` — polls POTA + SOTA every 2 minutes, merges,
  independent per-source stale-but-valid handling (one feed failing
  doesn't blank out the other).
- `internal/adif` — ADIF parser + worked-before `Index` (two O(1) lookup
  maps: worked-any-band, worked-this-band). `Index.Replace` is a full
  swap, not incremental.
- `internal/qrzlogbook` — syncs the `adif.Index` directly from QRZ's
  Logbook Data API on startup + hourly. **Not** the qrzlook sibling
  service's XML lookup API — a completely separate QRZ product with its
  own API key (QRZ account → Logbook → Settings → API).
- `main.go` — routing, config, and the only place that imports both a spot
  source (`cluster`/`parkspots`) and `adif`, via `decoratedSpot`/
  `decoratedParkSpot` wrapper structs (struct-embed the source spot type,
  add `worked_any`/`worked_band` json fields). Keeps `cluster` and
  `parkspots` decoupled from `adif` — they never need to know it exists.

Frontend mirrors this: one `web/js/<name>.js` per panel, each self-polling
its own `/api/...` endpoint and re-rendering its own DOM section
independently (`SpaceWx.start(ms)`, `Cluster.start(ms)`, etc., all kicked
off from `app.js`). No shared state between them beyond the DOM and
`HamMap`/`Qrz` globals.

## Non-obvious gotchas (found the hard way this session)

- **QRZ Logbook API HTML-escapes its ADIF payload.** `&lt;`/`&gt;` instead
  of `<`/`>`. Confirmed byte-for-byte against the live API — not
  documented anywhere obvious. `qrzlogbook.FetchAndReplace` calls
  `html.UnescapeString` before parsing; if you ever touch that code path,
  don't remove it, and don't trust "it returns ADIF" claims without
  checking raw bytes again.
- **QRZ Logbook API's `OPTION=MAX:0` means "zero records", not
  "unlimited."** Omitting `OPTION` entirely is what fetches everything.
  Also confirmed the hard way — `MAX:0` gave `RESULT=OK` with the correct
  total `COUNT` but no `ADIF=` field at all, which silently looked like
  "empty logbook" rather than an error.
- **ADIF field lengths are byte-exact, not delimiters.** `<call:5>W1AW`
  reads 5 bytes after the tag regardless of where the value "logically"
  ends — if the declared length doesn't match the actual value length,
  you get corrupted/truncated fields, not a parse error. Every test
  fixture in this repo has been wrong about this at least once; count
  characters carefully when writing new ADIF test data (`W1AW` is 4,
  `G8GDS` is 5, `VE2JCW` is 6...).
- **DX cluster login prompts arrive without a trailing newline.**
  `cluster.Client.login` reads byte-by-byte and checks for a `login:`/
  `call:` suffix rather than using a line-buffered reader, specifically
  because of this.
- **nginx on fimblefowl.co.uk uses `sites-enabled/*.conf`**, not a
  monolithic `nginx.conf` + `conf.d` split — that's the *old* server's
  layout (pre 2026-07-20 DNS cutover). Don't assume the old layout when
  writing nginx snippets; check `/etc/nginx/sites-enabled/fimblefowl.conf`
  directly.
- **Path prefix matters for relative vs absolute URLs.** nginx proxies
  `/shackboard/` with the prefix stripped, so shackboard's own JS must use
  relative paths (`fetch('api/spots')`, no leading slash) or requests hit
  the domain root and 404. The one deliberate exception is the QRZ
  callsign-lookup call (`fetch('/qrz/lookup/' + call)`), which is
  root-absolute on purpose — it targets the *sibling* qrzlook service's
  own top-level path on the same origin, not a shackboard path.

## Secrets

Never commit a real value for `SHACKBOARD_QRZ_LOGBOOK_KEY` — `shackboard.
service` in this repo must only ever contain the literal placeholder
`changeme`. The real key lives only in the *deployed* copy of that file on
the server (`~/.config/systemd/user/shackboard.service`), sourced from
`~/.secrets/quanglewangle.env` (`QRZ_LOGBOOK_API_KEY`) on the server host.
Same rule applied to `SHACKBOARD_WRITE_TOKEN` when that existed (removed —
see below).

## History worth knowing

A manual ADIF-upload endpoint (`POST /api/log`, token-protected) existed
briefly during development, before QRZ Logbook auto-sync replaced it
entirely per Peter's explicit choice. If you see references to it in old
commit messages or the plan file, that's why — it's gone from the code on
purpose, not an oversight. `nginx.conf.snippet` still has a note about a
leftover `client_max_body_size 16m;` on the live server from that era;
harmless, not worth reverting.

## Testing

`go test ./... -race`. Every `internal/*` package has table-driven tests
using real captured/confirmed data shapes (a real DX cluster spot line, a
real QRZ Logbook response shape, real POTA/SOTA JSON) rather than
invented fixtures, specifically because this codebase has already caught
multiple bugs where the *actual* external format differed from what
seemed like the obvious assumption. When adding a new external feed,
verify its real response shape live (`curl`) before writing the parser,
not after.

## Deploy

`./deploy.sh peter@fimblefowl.co.uk` — builds, scps, restarts the
`systemctl --user` service. Config/env var changes need manual editing of
`~/.config/systemd/user/shackboard.service` on the server (not something
`deploy.sh` touches) plus `systemctl --user daemon-reload`. nginx changes
need a `sudo`'d script handed to Peter to run himself — this account
can't sudo on that box.
