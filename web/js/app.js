(async function () {
  await HamMap.init();

  SpaceWx.start(5 * 60 * 1000);
  Cluster.start(15 * 1000);
  ParkSpots.start(30 * 1000);
  LogStatus.start(5 * 60 * 1000);

  const utcEl = document.getElementById('clock-utc');

  function pad(n) { return String(n).padStart(2, '0'); }

  function tick() {
    const now = new Date();
    utcEl.textContent = `${pad(now.getUTCHours())}:${pad(now.getUTCMinutes())}:${pad(now.getUTCSeconds())}Z`;
    HamMap.draw();
  }
  tick();
  setInterval(tick, 1000);

  const form = document.getElementById('qth-setup-form');
  const display = document.getElementById('qth-setup-display');
  const displayCall = document.getElementById('qth-setup-call');
  const input = document.getElementById('callsign-input');
  const saveBtn = document.getElementById('callsign-save');

  function showForm() {
    form.classList.remove('hidden');
    display.classList.add('hidden');
    input.focus();
  }

  function showDisplay(call) {
    displayCall.textContent = call;
    form.classList.add('hidden');
    display.classList.remove('hidden');
  }

  async function setHomeCallsign(call) {
    if (!call) return;
    localStorage.setItem('shackboard.callsign', call);
    const info = await Qrz.lookupAndPlot(call, 'home', '#4fb0ff');
    if (info) showDisplay(call);
  }

  saveBtn.addEventListener('click', () => setHomeCallsign(input.value.trim().toUpperCase()));
  input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') setHomeCallsign(input.value.trim().toUpperCase());
  });
  display.addEventListener('click', showForm);

  const savedCall = localStorage.getItem('shackboard.callsign');
  if (savedCall) {
    input.value = savedCall;
    setHomeCallsign(savedCall);
  }
})();
