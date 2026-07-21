// Shared band -> color mapping, grouped to match the propagation panel's
// band pairs (the hamqsl.com feed only reports conditions in these four
// pairs, e.g. "80m-40m" as one row) so the propagation table can double as
// a color key for band-coded spot markers/lines on the map. Bands outside
// these four HF pairs (160m, 60m, and everything VHF/UHF) fall back to a
// neutral color since the feed has no condition data for them anyway.

const BandColors = (() => {
  const GROUPS = [
    { label: '80m-40m', bands: ['80m', '40m'], color: '#c77dff' },
    { label: '30m-20m', bands: ['30m', '20m'], color: '#4fd8c7' },
    { label: '17m-15m', bands: ['17m', '15m'], color: '#ff8a65' },
    { label: '12m-10m', bands: ['12m', '10m'], color: '#f06292' },
  ];
  const FALLBACK = '#7c8ea3';

  const byBand = {};
  for (const g of GROUPS) {
    for (const b of g.bands) byBand[b] = g.color;
  }

  function forBand(band) {
    return byBand[(band || '').toLowerCase()] || FALLBACK;
  }

  function forGroupLabel(label) {
    const g = GROUPS.find(g => g.label === label);
    return g ? g.color : FALLBACK;
  }

  return { forBand, forGroupLabel, GROUPS, FALLBACK };
})();
