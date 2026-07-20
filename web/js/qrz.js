// Looks up a callsign via the sibling qrzlook service. Deliberately
// root-absolute ('/qrz/...') since it targets that service's existing
// top-level path on the same origin, unlike shackboard's own relative
// 'api/...' calls.

const Qrz = (() => {
  const popup = document.getElementById('callsign-popup');
  const popupBody = document.getElementById('callsign-popup-body');
  document.getElementById('callsign-popup-close').addEventListener('click', () => {
    popup.classList.add('hidden');
  });

  async function lookup(callsign) {
    const resp = await fetch('/qrz/lookup/' + encodeURIComponent(callsign));
    if (!resp.ok) return null;
    return resp.json();
  }

  function showPopup(info) {
    popupBody.innerHTML = `
      <dl>
        <dt>Callsign</dt><dd>${info.callsign || ''}</dd>
        <dt>Name</dt><dd>${[info.fname, info.lname].filter(Boolean).join(' ') || info.name || '—'}</dd>
        <dt>Location</dt><dd>${[info.city, info.state, info.country].filter(Boolean).join(', ') || '—'}</dd>
        <dt>Grid</dt><dd>${info.grid || '—'}</dd>
      </dl>
    `;
    popup.classList.remove('hidden');
  }

  function plotMarker(info, markerId, markerColor) {
    const lat = parseFloat(info.lat);
    const lon = parseFloat(info.lon);
    if (!Number.isNaN(lat) && !Number.isNaN(lon)) {
      HamMap.addMarker({ id: markerId, lat, lon, label: info.callsign, color: markerColor });
      HamMap.draw();
    }
  }

  async function lookupAndShow(callsign, markerId, markerColor) {
    const info = await lookup(callsign);
    if (!info) {
      popupBody.innerHTML = `<dl><dt>Callsign</dt><dd>${callsign}</dd></dl><p>Not found in QRZ.</p>`;
      popup.classList.remove('hidden');
      return null;
    }
    showPopup(info);
    plotMarker(info, markerId, markerColor);
    return info;
  }

  // Like lookupAndShow, but plots the marker without popping open the info
  // panel — for the home QTH, which resolves on every page load and
  // shouldn't force a popup shut each time.
  async function lookupAndPlot(callsign, markerId, markerColor) {
    const info = await lookup(callsign);
    if (info) plotMarker(info, markerId, markerColor);
    return info;
  }

  return { lookup, lookupAndShow, lookupAndPlot };
})();
