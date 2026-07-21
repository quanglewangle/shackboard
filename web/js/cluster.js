// Polls /api/spots and renders the DX spots panel.

const Cluster = (() => {
  const status = document.getElementById('spots-status');
  const body = document.getElementById('spots-body');

  // call -> {lat, lon} | null (null = looked up, not resolvable). Persists
  // for the life of the page so repeated polls don't re-hit the QRZ lookup
  // for calls already resolved — most polls only see a couple of new calls.
  const geoCache = new Map();
  let plottedCalls = new Set();

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
    const currentCalls = new Set(top.map(s => s.dx_call));

    for (const call of plottedCalls) {
      if (!currentCalls.has(call)) HamMap.removeMarker('spot-' + call);
    }
    plottedCalls = currentCalls;

    await Promise.all(top.map(s => resolveGeo(s.dx_call)));

    for (const s of top) {
      const geo = geoCache.get(s.dx_call);
      if (geo) {
        HamMap.addMarker({ id: 'spot-' + s.dx_call, lat: geo.lat, lon: geo.lon, label: s.dx_call, color: '#e0b23e' });
      }
    }
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
        <td>${s.time_utc}</td>
        <td>${s.band}</td>
        <td>${s.freq_khz.toFixed(1)}</td>
        <td class="dx-call" data-call="${s.dx_call}">${s.dx_call}</td>
        <td>${s.spotter}</td>
        <td class="comment">${s.comment}</td>
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
        // Same marker id scheme as plotSpotMarkers, so clicking a spot
        // that's already auto-plotted just re-confirms it rather than
        // creating a redundant second marker at the same position.
        Qrz.lookupAndShow(el.dataset.call, 'spot-' + el.dataset.call, '#e0b23e');
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
