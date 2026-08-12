// Polls /api/spots and renders the DX spots panel.

const Cluster = (() => {
  const status = document.getElementById('spots-status');
  const body = document.getElementById('spots-body');

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
        <td class="country">${s.country}</td>
        <td>${s.spotter}</td>
        <td class="comment">${s.comment}</td>
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
        Qrz.lookupAndShow(el.dataset.call, 'spot', '#e0b23e');
      });
    });

    // Auto-plot every spot with a resolved DXCC location (approximate —
    // entity center, not the actual station) as a small dot, distinct
    // from the brighter single pin a clicked callsign gets via Qrz.
    HamMap.setGroup('dxspots', data.spots
      .filter(s => s.country)
      .map(s => ({ id: 'dx-' + s.dx_call, lat: s.lat, lon: s.lon, color: '#e0b23e', r: 3 })));
    HamMap.draw();
  }

  async function poll() {
    try {
      const resp = await fetch('api/spots?limit=100');
      if (!resp.ok) return;
      render(await resp.json());
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
