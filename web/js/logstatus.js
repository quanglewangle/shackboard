// Polls /api/log and renders a small "Log: N QSOs, updated Xh ago" line —
// informs the worked-before highlighting shown in both the DX Spots and
// Parks & Summits tables, so it lives above both rather than inside either.

const LogStatus = (() => {
  const el = document.getElementById('log-status');

  function fmtAgo(isoString) {
    const then = new Date(isoString).getTime();
    const mins = Math.max(0, Math.round((Date.now() - then) / 60000));
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.round(mins / 60);
    if (hours < 48) return `${hours}h ago`;
    return `${Math.round(hours / 24)}d ago`;
  }

  function render(data) {
    if (!data.loaded) {
      el.textContent = 'Log: none loaded';
      return;
    }
    const age = data.synced_at ? `, synced ${fmtAgo(data.synced_at)}` : '';
    el.textContent = `Log: ${data.qso_count} QSOs${age}`;
  }

  async function poll() {
    try {
      const resp = await fetch('api/log');
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
