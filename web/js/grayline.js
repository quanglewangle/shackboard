// Low-precision NOAA solar-position formulas: subsolar point + terminator
// curve. No library, recomputed from the current time on every tick.

const Grayline = (() => {
  const RAD = Math.PI / 180;
  const DEG = 180 / Math.PI;

  function dayOfYearUTC(date) {
    const start = Date.UTC(date.getUTCFullYear(), 0, 1);
    return Math.floor((date.getTime() - start) / 86400000) + 1;
  }

  // Returns { lat, lon, decl } in degrees for the current subsolar point.
  function subsolarPoint(date) {
    const doy = dayOfYearUTC(date);
    const utcHours = date.getUTCHours() + date.getUTCMinutes() / 60 + date.getUTCSeconds() / 3600;

    const gamma = (2 * Math.PI / 365) * (doy - 1 + (utcHours - 12) / 24);

    const eqTimeMin = 229.18 * (
      0.000075
      + 0.001868 * Math.cos(gamma) - 0.032077 * Math.sin(gamma)
      - 0.014615 * Math.cos(2 * gamma) - 0.040849 * Math.sin(2 * gamma)
    );

    const declRad = (
      0.006918
      - 0.399912 * Math.cos(gamma) + 0.070257 * Math.sin(gamma)
      - 0.006758 * Math.cos(2 * gamma) + 0.000907 * Math.sin(2 * gamma)
      - 0.002697 * Math.cos(3 * gamma) + 0.00148 * Math.sin(3 * gamma)
    );

    let lon = -15 * (utcHours - 12 + eqTimeMin / 60);
    lon = ((lon + 180) % 360 + 360) % 360 - 180;

    return { lat: declRad * DEG, lon, decl: declRad };
  }

  // Returns an array of [lon, lat] points (degrees) tracing the terminator,
  // one per 2-degree longitude step.
  function terminatorPoints(date) {
    const { lon: subLon, decl } = subsolarPoint(date);
    const points = [];
    for (let lon = -180; lon <= 180; lon += 2) {
      let H = lon - subLon;
      H = ((H + 180) % 360 + 360) % 360 - 180;
      const Hrad = H * RAD;
      const lat = Math.atan2(-Math.cos(Hrad), Math.tan(decl)) * DEG;
      points.push([lon, lat]);
    }
    return points;
  }

  return { subsolarPoint, terminatorPoints };
})();
