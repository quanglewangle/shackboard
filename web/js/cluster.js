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

  // Marker ids currently plotted from the spot list, so a poll that drops
  // a spot (aged out of the buffer) removes its marker instead of leaving
  // a stale dot behind.
  let plottedMarkerIds = new Set();

  // Plots a DX + spotter marker and a connecting line for each of the 10
  // most recent spots with a resolved DXCC location. Country/lat/lon come
  // straight from the /api/spots response (server-side local cty.dat
  // lookup — see main.go's decorateSpots), not a live QRZ call per spot.
  function plotSpotMarkers(spots) {
    const top = spots.slice(0, 10);
    const currentIds = new Set();
    const lines = [];

    for (const s of top) {
      const color = BandColors.forBand(s.band);

      if (s.country) {
        const id = 'spot-' + s.dx_call;
        currentIds.add(id);
        HamMap.addMarker({ id, lat: s.lat, lon: s.lon, label: s.dx_call, color });
      }
      if (s.spotter_country) {
        const id = 'spotter-' + s.spotter;
        currentIds.add(id);
        HamMap.addMarker({ id, lat: s.spotter_lat, lon: s.spotter_lon, label: s.spotter, color });
      }
      if (s.country && s.spotter_country) {
        lines.push({
          lat1: s.spotter_lat, lon1: s.spotter_lon,
          lat2: s.lat, lon2: s.lon,
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
        <td style="color:${BandColors.forBand(s.band)}">${s.freq_khz.toFixed(1)}</td>
        <td class="dx-call" data-call="${escapeHtml(s.dx_call)}" data-band="${escapeHtml(s.band)}">${escapeHtml(s.dx_call)}</td>
        <td class="country">${escapeHtml(s.country)}</td>
        <td>${escapeHtml(s.spotter)}</td>
        <td class="comment">${escapeHtml(s.comment)}</td>
      </tr>
    `;
    }).join('');

    body.innerHTML = `
      <table class="spot-table">
        <thead><tr><th>UTC</th><th>Band</th><th>kHz</th><th>DX</th><th>Country</th><th>Spotter</th><th>Comment</th></tr></thead>
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
      plotSpotMarkers(data.spots);
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
