// Shared by overlay.html (game, plus the ranking the control panel can fade in), ranking.html and cams1.html / cams2.html.
// ranking.html and cams1/2.html pin their scene with <body data-scene>; cams1/2.html and chat1/2.html pin the camera count with <body data-camera>; commentary needs its own OBS scene, or the game capture would show through its camera holes.
const forced = document.body.dataset.scene, forcedCam = document.body.dataset.camera;
const $ = id => document.getElementById(id);
const sleep = ms => new Promise(r => setTimeout(r, ms));

// Seat cards are built once and updated in place, so changes can animate.
$('bar')?.insertAdjacentHTML('beforeend', [0, 1, 2, 3].map(i => `
  <div class="seat" id="seat${i}"><div class="wind"></div>
    <div><div class="stick"></div><div class="name"></div><div class="score"></div><div class="sub"><div class="diff"></div><div class="waits"></div></div></div>
    <div class="place"></div></div>`).join(''));

// Crossfade through images; rebuilt only when the list or speed changes.
function slideshow(el, srcs, secs) {
  if (!el) return;
  const key = srcs.join('|') + secs;
  if (el.key === key) return;
  el.key = key;
  clearInterval(el.timer);
  el.innerHTML = srcs.map((src, i) => `<img src="${esc(src)}" class="${i ? '' : 'on'}" alt="">`).join('');
  let i = 0;
  if (srcs.length > 1) el.timer = setInterval(() => {
    el.children[i].classList.remove('on');
    i = (i + 1) % srcs.length;
    el.children[i].classList.add('on');
  }, Math.max(2, secs) * 1000);
}
// Cut see-through holes in the full-screen scenes where the OBS camera sources sit under this page.
function punch() {
  for (const sc of document.querySelectorAll('.scene')) {
    const holes = [...sc.querySelectorAll('.hole')].filter(h => h.clientHeight > 0 && h.offsetParent).map(h => {
      const r = h.getBoundingClientRect(), x = r.left + h.clientLeft, y = r.top + h.clientTop;
      return `M${x} ${y}h${h.clientWidth}v${h.clientHeight}h${-h.clientWidth}Z`;
    });
    sc.style.clipPath = holes.length ? `path(evenodd, 'M0 0H1920V1080H0Z${holes.join('')}')` : '';
  }
}
const uploads = files => (files || []).map(f => `uploads/${encodeURIComponent(f)}`);

const pop = el => { el.classList.remove('pop'); void el.offsetWidth; el.classList.add('pop'); };
const setText = (el, t) => { if (el.textContent !== String(t)) { el.textContent = t; pop(el); } };

// Count the score up/down instead of jumping.
function tween(el, to, delay = 0, f = fmt) {
  const from = el.dataset.v === undefined ? to : +el.dataset.v;
  el.dataset.v = to;
  if (from === to) return void (el.textContent = f(to));
  const t0 = el.t0 = performance.now() + delay;
  const step = t => {
    if (el.t0 !== t0) return; // a newer change took over
    const k = Math.max(0, Math.min(1, (t - t0) / 900));
    el.textContent = f(Math.round((from + (to - from) * (1 - (1 - k) ** 3)) / 100) * 100);
    if (k < 1) requestAnimationFrame(step);
  };
  requestAnimationFrame(step);
  pop(el);
}

// Points changed: show "-8.000" or "+8.000", drain it to 0 while the score counts, then fade out.
const HOLD = 3200; // ms the change sits frozen before the score starts counting
function delta(el, n) {
  const sign = n < 0 ? '-' : '+';
  el.dataset.v = Math.abs(n);
  el.classList.add('on');
  el.classList.toggle('gain', n > 0);
  tween(el, 0, HOLD, v => sign + fmt(v));
  clearTimeout(el.hide);
  el.hide = setTimeout(() => el.classList.remove('on'), HOLD + 900 + 500);
}

// Standings that don't fit: hold at the top, scroll slowly to the end, hold, glide back. Forever.
async function autoScroll(view) {
  const list = view.firstElementChild;
  for (;;) {
    await sleep(8000);
    const d = list.offsetHeight - view.clientHeight;
    if (d <= 0) continue;
    const secs = d / 40; // 40 px per second
    list.style.transition = `transform ${secs}s linear`;
    list.style.transform = `translateY(${-d}px)`;
    await sleep(secs * 1000 + 5000);
    list.style.transition = '';
    list.style.transform = '';
    await sleep(1200);
  }
}
document.querySelectorAll('.view').forEach(autoScroll);

const row = (p, i) => `<li><span class="pos">${i + 1}º</span><span class="pn">${esc(p.name)}</span>` +
  `<span class="pt ${p.total < 0 ? 'neg' : ''}">${p.total > 0 ? '+' : ''}${fmt(p.total)}</span></li>`;
const setList = (el, html) => { if (el && el.last !== html) el.innerHTML = el.last = html; };

// Play the call only on a fresh declaration, never on the first state after a (re)load.
let prev;
function riichiCall(name) {
  const fx = $('riichiFx');
  fx.querySelector('span').textContent = name;
  fx.classList.remove('go'); void fx.offsetWidth; fx.classList.add('go');
}

const WIND_NAMES = { romaji: { 東: 'Ton', 南: 'Nan', 西: 'Sha', 北: 'Pei' }, pt: { 東: 'Leste', 南: 'Sul', 西: 'Oeste', 北: 'Norte' } };

function render(s) {
  if (forcedCam) s.camera = +forcedCam;
  const r = s.round, winds = seatWinds(r.num), place = placements(s.seats);
  const names = WIND_NAMES[s.windText], w = k => (names ? names[k] : k);
  document.body.classList.toggle('romaji', s.windText === 'romaji');
  document.body.classList.toggle('pt', s.windText === 'pt');

  const scene = forced || s.scene;
  document.body.classList.toggle('ranking', scene === 'ranking');
  document.body.classList.toggle('talk', scene === 'talk');
  document.body.classList.remove('cam0', 'cam1', 'cam2');
  document.body.classList.add(`cam${s.camera ?? 1}`);
  // Follow the camera boxes every frame while they animate (.6s), so the holes never lag behind them.
  const until = performance.now() + 800;
  const follow = () => { punch(); if (performance.now() < until) requestAnimationFrame(follow); };
  follow();

  if ($('bar')) {
    $('riichiFx').querySelector('b').textContent = s.riichiText === 'romaji' ? 'RIICHI' : 'リーチ';

    setText($('roundName'), names ? `${w(WINDS[r.wind])} ${r.num}` : `${WINDS[r.wind]}${r.num}局`);
    $('sticks').innerHTML = `Honba <b>${r.honba}</b> · Riichi <b>${r.riichi}</b>`;
    s.seats.forEach((x, i) => {
      const el = $(`seat${i}`), q = c => el.querySelector(c);
      el.classList.toggle('first', place[i] === 1);
      el.classList.toggle('dealer', winds[i] === '東');
      el.classList.toggle('riichi', !!x.riichi);
      if (prev && x.riichi && !prev.seats[i].riichi) riichiCall(x.name);
      setText(q('.wind'), winds[i]);
      q('.name').textContent = x.name || '—';
      const old = q('.score').dataset.v, d = old === undefined ? 0 : x.score - old;
      if (d) delta(q('.diff'), d);
      tween(q('.score'), x.score, d ? HOLD : 0);
      setText(q('.place'), `${place[i]}º`);
      const waits = (Array.isArray(x.waits) ? x.waits : []).map(t => `<img src="${tileSrc(t)}" alt="${t}">`).join('');
      if (q('.waits').innerHTML !== waits) q('.waits').innerHTML = waits;
    });
  }

  const all = (sel, text) => document.querySelectorAll(sel).forEach(el => (el.textContent = text));
  all('#title, .title', s.title);
  all('#subtitle, .subtitle', s.subtitle);
  const mic = t => t || '';
  const both = mic([s.casters, s.casters2].filter(Boolean).join(' · '));
  all('#casters, .camCasters, .casters', s.camera === 2 ? mic(s.casters) : both);
  all('.camCasters2, .casters2', mic(s.casters2));
  document.body.classList.toggle('noPlayers', !s.players.length);
  const html = standings(s.players).map(row).join('');
  setList($('top'), html);
  setList($('all'), html);
  slideshow($('sponsors'), ['assets/logo.png', ...uploads(s.sponsors)], s.rotateSecs ?? 8);
  const banners = uploads(s.banners);
  slideshow($('banner'), banners.length ? banners : ['assets/nanda.png'], s.rotateSecs ?? 8);
  slideshow($('banner2'), banners.length ? banners : ['assets/nanda.png'], s.rotateSecs ?? 8);
  prev = s;
}

new EventSource('/api/events').onmessage = e => render(JSON.parse(e.data));
