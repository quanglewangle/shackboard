// Renders the world map: embedded vector coastlines (Canvas Path2D) plus a
// grayline/terminator overlay, QRZ-derived markers, and great-circle lines
// from the home marker to every other marker. Static equirectangular
// projection — this is a fixed full-disk display, not pannable/zoomable, so
// no tile library is needed.

const HamMap = (() => {
  const canvas = document.getElementById('map');
  const ctx = canvas.getContext('2d');

  let landPaths = [];
  let markers = []; // { lat, lon, label, color }

  function resize() {
    const rect = canvas.parentElement.getBoundingClientRect();
    const dpr = window.devicePixelRatio || 1;
    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    canvas.style.width = rect.width + 'px';
    canvas.style.height = rect.height + 'px';
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  }

  function project(lon, lat, w, h) {
    return [(lon + 180) / 360 * w, (90 - lat) / 180 * h];
  }

  function ringToPath(ring, w, h) {
    const path = new Path2D();
    ring.forEach(([lon, lat], i) => {
      const [x, y] = project(lon, lat, w, h);
      if (i === 0) path.moveTo(x, y); else path.lineTo(x, y);
    });
    path.closePath();
    return path;
  }

  async function loadCoastlines() {
    const resp = await fetch('data/ne-110m-land.geojson');
    const geo = await resp.json();
    // Cache raw rings; paths are rebuilt on resize since projection depends
    // on canvas size.
    HamMap._rings = [];
    for (const feature of geo.features) {
      const geom = feature.geometry;
      const polys = geom.type === 'Polygon' ? [geom.coordinates] : geom.coordinates;
      for (const poly of polys) {
        for (const ring of poly) {
          HamMap._rings.push(ring);
        }
      }
    }
  }

  function rebuildLandPaths(w, h) {
    landPaths = (HamMap._rings || []).map(ring => ringToPath(ring, w, h));
  }

  // Points along the great-circle path from (lat1,lon1) to (lat2,lon2),
  // via spherical interpolation (slerp) between the two unit vectors —
  // standard intermediate-point-on-great-circle formula.
  function greatCirclePoints(lat1, lon1, lat2, lon2, steps) {
    const toRad = d => d * Math.PI / 180;
    const toDeg = r => r * 180 / Math.PI;
    const phi1 = toRad(lat1), lam1 = toRad(lon1);
    const phi2 = toRad(lat2), lam2 = toRad(lon2);

    const delta = 2 * Math.asin(Math.sqrt(
      Math.sin((phi2 - phi1) / 2) ** 2 +
      Math.cos(phi1) * Math.cos(phi2) * Math.sin((lam2 - lam1) / 2) ** 2
    ));
    if (delta === 0 || Number.isNaN(delta)) return [[lon1, lat1]];

    const points = [];
    for (let i = 0; i <= steps; i++) {
      const f = i / steps;
      const a = Math.sin((1 - f) * delta) / Math.sin(delta);
      const b = Math.sin(f * delta) / Math.sin(delta);
      const x = a * Math.cos(phi1) * Math.cos(lam1) + b * Math.cos(phi2) * Math.cos(lam2);
      const y = a * Math.cos(phi1) * Math.sin(lam1) + b * Math.cos(phi2) * Math.sin(lam2);
      const z = a * Math.sin(phi1) + b * Math.sin(phi2);
      const phi = Math.atan2(z, Math.sqrt(x * x + y * y));
      const lam = Math.atan2(y, x);
      points.push([toDeg(lam), toDeg(phi)]);
    }
    return points;
  }

  // A great-circle line drawn on a flat equirectangular map has to break
  // into a fresh subpath wherever it crosses the antimeridian, or it draws
  // a bogus line straight across the map instead of exiting one edge and
  // re-entering the other.
  function greatCirclePath(lat1, lon1, lat2, lon2, w, h) {
    const points = greatCirclePoints(lat1, lon1, lat2, lon2, 64);
    const path = new Path2D();
    let prevLon = null;
    points.forEach(([lon, lat], i) => {
      const [x, y] = project(lon, lat, w, h);
      if (i === 0 || (prevLon !== null && Math.abs(lon - prevLon) > 180)) {
        path.moveTo(x, y);
      } else {
        path.lineTo(x, y);
      }
      prevLon = lon;
    });
    return path;
  }

  function nightPolygonPath(date, w, h) {
    const points = Grayline.terminatorPoints(date);
    const { lat: subLat } = Grayline.subsolarPoint(date);
    const darkPole = subLat >= 0 ? -90 : 90;

    const path = new Path2D();
    points.forEach(([lon, lat], i) => {
      const [x, y] = project(lon, lat, w, h);
      if (i === 0) path.moveTo(x, y); else path.lineTo(x, y);
    });
    let [x, y] = project(180, darkPole, w, h);
    path.lineTo(x, y);
    [x, y] = project(-180, darkPole, w, h);
    path.lineTo(x, y);
    path.closePath();
    return path;
  }

  function draw() {
    const w = canvas.clientWidth;
    const h = canvas.clientHeight;
    if (w === 0 || h === 0) return;

    ctx.clearRect(0, 0, w, h);

    ctx.fillStyle = '#1c5480';
    ctx.fillRect(0, 0, w, h);

    ctx.fillStyle = '#4a7a4a';
    ctx.strokeStyle = '#7fae7f';
    ctx.lineWidth = 1;
    for (const p of landPaths) {
      ctx.fill(p);
      ctx.stroke(p);
    }

    ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
    ctx.fill(nightPolygonPath(new Date(), w, h));

    ctx.strokeStyle = 'rgba(180, 180, 180, 0.35)';
    ctx.beginPath();
    ctx.moveTo(w / 2, 0);
    ctx.lineTo(w / 2, h);
    ctx.moveTo(0, h / 2);
    ctx.lineTo(w, h / 2);
    ctx.stroke();

    const home = markers.find(m => m.id === 'home');
    if (home) {
      ctx.strokeStyle = 'rgba(224, 178, 62, 0.5)';
      ctx.lineWidth = 1;
      for (const m of markers) {
        if (m.id === 'home') continue;
        ctx.stroke(greatCirclePath(home.lat, home.lon, m.lat, m.lon, w, h));
      }
    }

    for (const m of markers) {
      const [x, y] = project(m.lon, m.lat, w, h);
      ctx.beginPath();
      ctx.arc(x, y, 5, 0, Math.PI * 2);
      ctx.fillStyle = m.color || '#4fb0ff';
      ctx.fill();
      ctx.strokeStyle = '#0b0f14';
      ctx.lineWidth = 1.5;
      ctx.stroke();
      if (m.label) {
        ctx.fillStyle = '#d8e1ea';
        ctx.font = '11px sans-serif';
        ctx.fillText(m.label, x + 8, y + 4);
      }
    }
  }

  function setMarkers(next) {
    markers = next;
  }

  function addMarker(marker) {
    markers = markers.filter(m => m.id !== marker.id);
    markers.push(marker);
  }

  function removeMarker(id) {
    markers = markers.filter(m => m.id !== id);
  }

  async function init() {
    await loadCoastlines();
    // ResizeObserver (rather than a manual call + window 'resize' listener)
    // because its first callback is guaranteed to fire after layout has
    // actually settled — a manual resize() call here would run mid-parse,
    // before the flex/aspect-ratio layout of #map-panel has stabilized,
    // and capture a stale box size. It also naturally catches later size
    // changes that aren't triggered by a window resize (e.g. a sibling
    // panel's content changing #map-panel's stretched size).
    const ro = new ResizeObserver(() => {
      resize();
      rebuildLandPaths(canvas.clientWidth, canvas.clientHeight);
      draw();
    });
    ro.observe(canvas.parentElement);
  }

  return { init, draw, setMarkers, addMarker, removeMarker };
})();
