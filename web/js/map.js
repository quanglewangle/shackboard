// Renders the world map: embedded vector coastlines (Canvas Path2D) plus a
// grayline/terminator overlay and QRZ-derived markers. Static equirectangular
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

  return { init, draw, setMarkers, addMarker };
})();
