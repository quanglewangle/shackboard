// Renders the world map: embedded vector coastlines (Canvas Path2D) plus a
// grayline/terminator overlay and QRZ-derived markers. Static equirectangular
// projection — this is a fixed full-disk display, not pannable/zoomable, so
// no tile library is needed.

const HamMap = (() => {
  const canvas = document.getElementById('map');
  const ctx = canvas.getContext('2d');

  let landPaths = [];
  let markers = []; // { lat, lon, label, color }
  let lines = []; // { lat1, lon1, lat2, lon2, color }

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

  // Interpolates `steps` points along the great-circle path between two
  // lat/lon points (degrees), via spherical linear interpolation (slerp)
  // of their unit vectors — the shortest path on a sphere, not a straight
  // line on the equirectangular projection.
  function greatCirclePoints(lat1, lon1, lat2, lon2, steps) {
    const rad = Math.PI / 180, deg = 180 / Math.PI;
    const p1 = lat1 * rad, l1 = lon1 * rad, p2 = lat2 * rad, l2 = lon2 * rad;
    const x1 = Math.cos(p1) * Math.cos(l1), y1 = Math.cos(p1) * Math.sin(l1), z1 = Math.sin(p1);
    const x2 = Math.cos(p2) * Math.cos(l2), y2 = Math.cos(p2) * Math.sin(l2), z2 = Math.sin(p2);
    const dot = Math.max(-1, Math.min(1, x1 * x2 + y1 * y2 + z1 * z2));
    const d = Math.acos(dot);
    if (d < 1e-9) return [[lon1, lat1]];

    const points = [];
    for (let i = 0; i <= steps; i++) {
      const f = i / steps;
      const a = Math.sin((1 - f) * d) / Math.sin(d);
      const b = Math.sin(f * d) / Math.sin(d);
      const x = a * x1 + b * x2, y = a * y1 + b * y2, z = a * z1 + b * z2;
      points.push([Math.atan2(y, x) * deg, Math.atan2(z, Math.sqrt(x * x + y * y)) * deg]);
    }
    return points;
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

    for (const ln of lines) {
      const pts = greatCirclePoints(ln.lat1, ln.lon1, ln.lat2, ln.lon2, 32);
      ctx.beginPath();
      let prevX = null;
      for (const [lon, lat] of pts) {
        const [x, y] = project(lon, lat, w, h);
        // A great-circle path crossing the antimeridian projects as a
        // huge jump in x on this flat map; start a fresh subpath instead
        // of drawing a line straight across the whole width.
        if (prevX === null || Math.abs(x - prevX) > w / 2) {
          ctx.moveTo(x, y);
        } else {
          ctx.lineTo(x, y);
        }
        prevX = x;
      }
      ctx.strokeStyle = ln.color || 'rgba(224, 178, 62, 0.4)';
      ctx.lineWidth = 1;
      ctx.stroke();
    }

    for (const m of markers) {
      const [x, y] = project(m.lon, m.lat, w, h);
      ctx.beginPath();
      ctx.arc(x, y, m.r || 5, 0, Math.PI * 2);
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

  // Replaces every marker tagged with `group` in one go, leaving markers
  // added via addMarker (home QTH, click-to-plot) or other groups alone.
  // Used for bulk-plotting an entire spot list that refreshes on a poll
  // timer, where individual ids would otherwise accumulate stale dots.
  function setGroup(group, list) {
    markers = markers.filter(m => m.group !== group)
      .concat(list.map(m => ({ ...m, group })));
  }

  // Same idea as setGroup, but for lines.
  function setLineGroup(group, list) {
    lines = lines.filter(l => l.group !== group)
      .concat(list.map(l => ({ ...l, group })));
  }

  async function init() {
    resize();
    await loadCoastlines();
    rebuildLandPaths(canvas.clientWidth, canvas.clientHeight);
    window.addEventListener('resize', () => {
      resize();
      rebuildLandPaths(canvas.clientWidth, canvas.clientHeight);
      draw();
    });
    draw();
  }

  return { init, draw, setMarkers, addMarker, setGroup, setLineGroup };
})();
