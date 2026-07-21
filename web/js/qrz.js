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

  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, c => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
    }[c]));
  }

  async function lookup(callsign) {
    const resp = await fetch('/qrz/lookup/' + encodeURIComponent(callsign));
    if (!resp.ok) return null;
    return resp.json();
  }

  // Shackboard's own worked-before index — relative path (unlike the
  // root-absolute qrzlook lookup above), since this is shackboard's own
  // endpoint rather than the sibling service's.
  async function fetchContacts(callsign) {
    try {
      const resp = await fetch('api/log/contacts/' + encodeURIComponent(callsign));
      if (!resp.ok) return null;
      return await resp.json();
    } catch (e) {
      return null;
    }
  }

  function contactLine(c) {
    return [c.date, c.band, c.mode].filter(Boolean).map(escapeHtml).join(' ');
  }

  function contactsHtml(data) {
    if (!data || !data.loaded) return '<p class="prior-contact">Log sync not enabled.</p>';
    const contacts = data.contacts || [];
    if (contacts.length === 0) return '<p class="prior-contact">Not previously worked.</p>';

    const [first, ...rest] = contacts;
    let html = `<p class="prior-contact">Worked before: ${contactLine(first)}`;
    if (rest.length > 0) {
      html += ` <a href="#" class="more-contacts">+${rest.length} more</a>`;
    }
    html += '</p>';
    if (rest.length > 0) {
      html += '<ul class="more-contacts-list hidden">' +
        rest.map(c => `<li>${contactLine(c)}</li>`).join('') + '</ul>';
    }
    return html;
  }

  function showPopup(info, contacts) {
    popupBody.innerHTML = `
      <dl>
        <dt>Callsign</dt><dd>${escapeHtml(info.callsign || '')}</dd>
        <dt>Name</dt><dd>${escapeHtml([info.fname, info.lname].filter(Boolean).join(' ') || info.name || '—')}</dd>
        <dt>Location</dt><dd>${escapeHtml([info.city, info.state, info.country].filter(Boolean).join(', ') || '—')}</dd>
        <dt>Grid</dt><dd>${escapeHtml(info.grid || '—')}</dd>
      </dl>
      ${contacts ? contactsHtml(contacts) : ''}
    `;
    popup.classList.remove('hidden');

    const moreLink = popupBody.querySelector('.more-contacts');
    if (moreLink) {
      moreLink.addEventListener('click', (e) => {
        e.preventDefault();
        popupBody.querySelector('.more-contacts-list').classList.remove('hidden');
        moreLink.classList.add('hidden');
      });
    }
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
    const [info, contacts] = await Promise.all([lookup(callsign), fetchContacts(callsign)]);
    if (!info) {
      popupBody.innerHTML = `<dl><dt>Callsign</dt><dd>${escapeHtml(callsign)}</dd></dl><p>Not found in QRZ.</p>`;
      popup.classList.remove('hidden');
      return null;
    }
    showPopup(info, contacts);
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
