# Using shackboard

shackboard is a shack dashboard, live at **https://fimblefowl.co.uk/shackboard/**.
No login, no setup beyond entering your callsign once.

## The map

The world map shows the current day/night terminator (grayline), redrawn
every second. The UTC and local clocks in the top-left update live. The
horizontal/vertical lines are the equator and prime meridian, for
orientation.

## Set your callsign

Bottom-left of the map: type your callsign and click **Set**. This looks
you up via QRZ, plots a marker for your home QTH on the map, and remembers
the callsign in your browser (not the server) so it's still set next time
you load the page on that device. Click the "Callsign: G8GDS (change)"
button that appears afterward if you want to change it.

## Propagation panel

Per-band conditions (80m-40m, 30m-20m, 17m-15m, 12m-10m), each as Good /
Fair / Poor for day and night separately. This comes from the N0NBH solar
data feed and refreshes hourly — the timestamp under the table shows when
it was last pulled.

## DX Spots panel

Live spots from the DX cluster (`dxc.ve7cc.net`), refreshing every 15
seconds. Click any callsign to look it up on QRZ — a popup shows their
name/location/grid, and a marker gets plotted on the map. If a row looks
dimmed with a red stripe on the left, that means you've worked that exact
callsign on that exact band before (see "Worked before" below). A row
that's just dimmed without the stripe means you've worked that callsign,
but on a *different* band — might still be worth working.

## Parks & Summits panel

Live POTA (Parks on the Air) and SOTA (Summits on the Air) activator
spots, merged into one list, refreshing every 30 seconds. Green "POTA" /
blue "SOTA" badges tell them apart. Same click-to-lookup and worked-before
highlighting as DX Spots.

## Worked before

The "Log: N QSOs, synced Xh ago" line above the panels shows the state of
your worked-before log. This is synced **automatically** from your QRZ
Logbook — there's nothing to upload or configure. It refreshes on its own
roughly every hour; if you log a new QSO on QRZ, it can take up to that
long to show up here. If it ever says "Log: none loaded", that means the
server-side QRZ Logbook sync isn't configured — that's a server config
issue, not something to fix from the browser.

## What this doesn't do (yet)

No satellite tracking, no VOACAP point-to-point propagation predictions,
no sun/moon detail panel, no NCDXF beacon monitor, no weather. See the
project's `CLAUDE.md`/commit history if any of these come up later — a
few were scoped out deliberately, not forgotten.
