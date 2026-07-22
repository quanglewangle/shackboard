// Polls /api/spots and renders the DX spots panel.

const Cluster = (() => {
  const status = document.getElementById('spots-status');
  const body = document.getElementById('spots-body');

  // Spot fields (comment especially) are free text from a public DX
  // cluster feed — anyone spotting can put arbitrary characters in there,
  // e.g. a real captured comment once contained a literal "<TR>" (a grid
  // exchange convention), which a browser reads as an actual <tr> tag if
  // inserted unescaped into this table's innerHTML.
  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, c => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
    }[c]));
  }

  // call -> {lat, lon} | null (null = looked up, not resolvable). Persists
  // for the life of the page so repeated polls don't re-hit the QRZ lookup
  // for calls already resolved — most polls only see a couple of new calls.
  const geoCache = new Map();
  let plottedMarkerIds = new Set();

  async function resolveGeo(call) {
    if (geoCache.has(call)) return geoCache.get(call);
    const info = await Qrz.lookup(call);
    let geo = null;
    if (info) {
      const lat = parseFloat(info.lat);
      const lon = parseFloat(info.lon);
      if (!Number.isNaN(lat) && !Number.isNaN(lon)) geo = { lat, lon };
    }
    geoCache.set(call, geo);
    return geo;
  }

  async function plotSpotMarkers(spots) {
    const top = spots.slice(0, 10);

    // Lines are spotter -> DX station (who actually heard whom), so both
    // ends need resolving, not just the DX call — and both ends get a
    // marker plotted too, or a line would appear to dangle at whichever
    // end has no dot/callsign to identify it.
    await Promise.all(top.flatMap(s => [resolveGeo(s.dx_call), resolveGeo(s.spotter)]));

    const currentIds = new Set();
    const lines = [];
    for (const s of top) {
      const dxGeo = geoCache.get(s.dx_call);
      const spotterGeo = geoCache.get(s.spotter);
      const color = BandColors.forBand(s.band);

      if (dxGeo) {
        const id = 'spot-' + s.dx_call;
        currentIds.add(id);
        HamMap.addMarker({ id, lat: dxGeo.lat, lon: dxGeo.lon, label: s.dx_call, color });
      }
      if (spotterGeo) {
        const id = 'spotter-' + s.spotter;
        currentIds.add(id);
        HamMap.addMarker({ id, lat: spotterGeo.lat, lon: spotterGeo.lon, label: s.spotter, color });
      }
      if (dxGeo && spotterGeo) {
        lines.push({
          lat1: spotterGeo.lat, lon1: spotterGeo.lon,
          lat2: dxGeo.lat, lon2: dxGeo.lon,
          color,
        });
      }
    }

    for (const id of plottedMarkerIds) {
      if (!currentIds.has(id)) HamMap.removeMarker(id);
    }
    plottedMarkerIds = currentIds;

    HamMap.setLines(lines);
    HamMap.draw();
  }

  function render(data) {
    status.textContent = data.cluster_connected
      ? `Connected to ${data.cluster_host}`
      : `Disconnected from ${data.cluster_host} — reconnecting`;
    status.className = data.cluster_connected ? 'connected' : 'disconnected';

    const rows = data.spots.map(s => {
      const workedClass = s.worked_band ? 'worked-band' : (s.worked_any ? 'worked-any' : '');
      return `
      <tr class="${workedClass}">
        <td>${escapeHtml(s.time_utc)}</td>
        <td>${escapeHtml(s.band)}</td>
        <td>${s.freq_khz.toFixed(1)}</td>
        <td class="dx-call" data-call="${escapeHtml(s.dx_call)}" data-band="${escapeHtml(s.band)}">${escapeHtml(s.dx_call)}</td>
        <td>${escapeHtml(s.spotter)}</td>
        <td class="comment">${escapeHtml(s.comment)}</td>
      </tr>
    `;
    }).join('');

    body.innerHTML = `
      <table class="spot-table">
        <thead><tr><th>UTC</th><th>Band</th><th>kHz</th><th>DX</th><th>Spotter</th><th>Comment</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>
    `;

    body.querySelectorAll('.dx-call').forEach(el => {
      el.addEventListener('click', () => {
        // Same marker id scheme and band color as plotSpotMarkers, so
        // clicking a spot that's already auto-plotted just re-confirms it
        // rather than creating a redundant, differently-colored marker at
        // the same position.
        Qrz.lookupAndShow(el.dataset.call, 'spot-' + el.dataset.call, BandColors.forBand(el.dataset.band));
      });
    });
  }

  async function poll() {
    try {
      const resp = await fetch('api/spots?limit=100');
      if (!resp.ok) return;
      const data = await resp.json();
      render(data);
      await plotSpotMarkers(data.spots);
    } catch (e) {
      // transient — next poll will retry
    }
  }

  function start(intervalMs) {
    poll();
    setInterval(poll, intervalMs);
  }

  return { start };
})();
