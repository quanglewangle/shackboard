// Polls /api/park-spots and renders the Parks & Summits panel. Mirrors
// cluster.js's structure; reuses Qrz.lookupAndShow for click-to-lookup,
// same as the DX spots table.

const ParkSpots = (() => {
  const status = document.getElementById('parks-status');
  const body = document.getElementById('parks-body');

  function render(data) {
    const parts = [];
    if (!data.pota_ok) parts.push('POTA down');
    if (!data.sota_ok) parts.push('SOTA down');
    status.textContent = parts.length ? parts.join(' · ') : 'POTA + SOTA ok';
    status.className = parts.length ? 'disconnected' : 'connected';

    const rows = data.spots.map(s => {
      const workedClass = s.worked_band ? 'worked-band' : (s.worked_any ? 'worked-any' : '');
      const badgeClass = s.program === 'POTA' ? 'prog-pota' : 'prog-sota';
      return `
      <tr class="${workedClass}">
        <td><span class="prog-badge ${badgeClass}">${s.program}</span></td>
        <td>${s.band}</td>
        <td>${s.freq_khz.toFixed(1)}</td>
        <td class="dx-call" data-call="${s.activator}">${s.activator}</td>
        <td>${s.reference}</td>
        <td class="ref-name" title="${s.ref_name}">${s.ref_name}</td>
      </tr>
    `;
    }).join('');

    body.innerHTML = `
      <table class="spot-table">
        <thead><tr><th>Prog</th><th>Band</th><th>kHz</th><th>Activator</th><th>Ref</th><th>Name</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>
    `;

    body.querySelectorAll('.dx-call').forEach(el => {
      el.addEventListener('click', () => {
        Qrz.lookupAndShow(el.dataset.call, 'parkspot', '#e0b23e');
      });
    });
  }

  async function poll() {
    try {
      const resp = await fetch('api/park-spots');
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
