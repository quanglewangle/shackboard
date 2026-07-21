// Polls /api/spacewx and renders the per-band propagation condition panel.

const SpaceWx = (() => {
  const body = document.getElementById('propagation-body');

  function condClass(cond) {
    switch ((cond || '').toLowerCase()) {
      case 'good': return 'cond-good';
      case 'fair': return 'cond-fair';
      case 'poor': return 'cond-poor';
      default: return '';
    }
  }

  function render(data) {
    const rows = (data.bands || []).map(b => `
      <tr>
        <td><span class="band-swatch" style="background:${BandColors.forGroupLabel(b.band)}"></span>${b.band}</td>
        <td class="${condClass(b.day)}">${b.day || '?'}</td>
        <td class="${condClass(b.night)}">${b.night || '?'}</td>
      </tr>
    `).join('');

    body.innerHTML = `
      <table>
        <thead><tr><th>Band</th><th>Day</th><th>Night</th></tr></thead>
        <tbody>${rows}</tbody>
      </table>
      <div class="updated">Feed updated ${data.updated_utc || '?'} UTC</div>
    `;
  }

  async function poll() {
    try {
      const resp = await fetch('api/spacewx');
      if (!resp.ok) {
        body.textContent = 'Propagation data not yet available.';
        return;
      }
      render(await resp.json());
    } catch (e) {
      body.textContent = 'Failed to load propagation data.';
    }
  }

  function start(intervalMs) {
    poll();
    setInterval(poll, intervalMs);
  }

  return { start };
})();
