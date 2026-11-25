// Shared by the dashboard pages (control.html, media.html): nav bar, UI language, live state sync.
// Load after the page markup and common.js. Pages set `onFill` to redraw their own parts and call applyLang() once built.
const $ = id => document.getElementById(id);

// UI language is per operator (saved in this browser), not part of the shared overlay state.
const STR = {
  pt: {
    navGame: 'Jogo', navMedia: 'Banners e patrocinadores',
    sources: 'Fontes de navegador do OBS (1920×1080)', inGame: 'Jogo', breakScreen: 'Intervalo / classificação',
    broadcast: 'Transmissão', title: 'Título', subtitle: 'Subtítulo (fase / mesa)', casters: 'Narrador(es) — câmera 1', casters2: 'Narrador(es) — câmera 2',
    scene: 'Cena', talkScreen: 'Narração (câmeras + chat)',
    camera: 'Câmera dos narradores', hidden: 'Oculta', oneCam: '1 câmera', twoCams: '2 câmeras', windText: 'Vento da rodada', windJa: 'Japonês (東1局)', windPt: 'Português (Leste 1)', riichiText: 'Chamada de riichi', riichiJa: 'Japonês (リーチ)',
    round: 'Rodada', wind: 'Vento', hand: 'Mão', riichi: 'Palitos de riichi', startScore: 'Pontos iniciais',
    east: '東 Leste', south: '南 Sul', west: '西 Oeste', north: '北 Norte',
    next: 'Próxima mão', nextTip: 'O dealer passa, honba volta a 0', renchan: 'Dealer repete', renchanTip: '+1 honba',
    resetHand: 'Desfazer mão', resetHandTip: 'Volta ao início desta mão (de novo logo em seguida: mais uma mão)',
    resetRound: 'Novo jogo', resetRoundTip: '東1, palitos 0, pontos iniciais para todos',
    table: 'Mesa', tableHelp: 'Jogadores na ordem inicial (primeiro = Leste no 東1). O dealer fica destacado.',
    standings: 'Classificação do torneio', standingsHelp: 'Um jogador por linha: <code>Nome pontos</code> (ex.: <code>Ana Souza 45,5</code>)',
    player: 'Jogador', waits: 'Espera', waitsTitle: 'Espera (tenpai)', clear: 'Limpar', done: 'OK', riichiBtn: 'Riichi', saved: '✔ salvo', offline: '✖ servidor inacessível',
    noSnap: 'Nenhum início de mão salvo ainda. Use "Próxima mão" ou "Dealer repete" primeiro.',
    confirmHand: 'Desfazer todas as alterações desde o início desta mão?',
    confirmRound: 'Começar um novo jogo? Rodada volta para 東1, palitos para 0, pontos para {n}.',
    winBtn: '🀄 Vitória / empate', draw: 'Empate', tenpai: 'Quem estava tenpai', winTitle: 'Vitória', winner: 'Quem ganhou (no ron dá para marcar mais de um: ron duplo/triplo)', dealtIn: 'Quem pagou (ron)',
    points: 'Valor da mão de cada ganhador (valor do ron; no tsumo é dividido)', preview: 'Resultado', cancel: 'Cancelar', apply: 'Aplicar e ir para a próxima mão',
    banners: 'Banners (tela de intervalo, no lugar do mascote)', sponsors: 'Patrocinadores (canto inferior direito do jogo)',
    rotate: 'Troca a cada (segundos)', upload: 'Adicionar imagens (png, jpg, webp, gif — até 10 MB)',
    empty: 'Nenhuma imagem ainda.', uploadFail: 'Falha ao enviar',
  },
  en: {
    navGame: 'Game', navMedia: 'Banners & sponsors',
    sources: 'OBS browser sources (1920×1080)', inGame: 'In-game', breakScreen: 'Break / ranking',
    broadcast: 'Broadcast', title: 'Title', subtitle: 'Subtitle (stage / table)', casters: 'Commentator(s) — camera 1', casters2: 'Commentator(s) — camera 2',
    scene: 'Scene', talkScreen: 'Commentary (cameras + chat)',
    camera: 'Commentator camera', hidden: 'Hidden', oneCam: '1 camera', twoCams: '2 cameras', windText: 'Round wind', windJa: 'Japanese (東1局)', windPt: 'Portuguese (Leste 1)', riichiText: 'Riichi call', riichiJa: 'Japanese (リーチ)',
    round: 'Round', wind: 'Wind', hand: 'Hand', riichi: 'Riichi sticks', startScore: 'Starting points',
    east: '東 East', south: '南 South', west: '西 West', north: '北 North',
    next: 'Next hand', nextTip: 'Dealer moves, honba back to 0', renchan: 'Dealer repeats', renchanTip: '+1 honba',
    resetHand: 'Undo hand', resetHandTip: 'Back to the start of this hand (again right after: one more hand back)',
    resetRound: 'New game', resetRoundTip: '東1, sticks 0, starting points for everyone',
    table: 'Table', tableHelp: 'Seats in starting order (first = East at 東1). The dealer is highlighted.',
    standings: 'Tournament standings', standingsHelp: 'One player per line: <code>Name points</code> (e.g. <code>Ana Souza 45,5</code>)',
    player: 'Player', waits: 'Waits', waitsTitle: 'Waits (tenpai)', clear: 'Clear', done: 'OK', riichiBtn: 'Riichi', saved: '✔ saved', offline: '✖ server not reachable',
    noSnap: 'No hand start saved yet. Use "Next hand" or "Dealer repeats" first.',
    confirmHand: 'Undo all changes since the start of this hand?',
    confirmRound: 'Start a new game? Round goes to 東1, sticks to 0, scores to {n}.',
    winBtn: '🀄 Win / draw', draw: 'Draw', tenpai: 'Who was tenpai', winTitle: 'Win', winner: 'Winner (ron: tick more than one for double/triple ron)', dealtIn: 'Dealt in (ron)',
    points: 'Hand value per winner (ron value; tsumo splits it)', preview: 'Result', cancel: 'Cancel', apply: 'Apply and go to next hand',
    banners: 'Banners (break screen, in place of the mascot)', sponsors: 'Sponsors (bottom-right corner in game)',
    rotate: 'Change every (seconds)', upload: 'Add images (png, jpg, webp, gif — up to 10 MB)',
    empty: 'No images yet.', uploadFail: 'Upload failed',
  },
};
let lang = localStorage.lang || (navigator.language.startsWith('pt') ? 'pt' : 'en');
const t = k => STR[lang][k];

document.body.insertAdjacentHTML('afterbegin', `<nav>
  <img src="assets/logo.png" alt="CZS">
  <a href="/" data-i18n="navGame"></a><a href="/media.html" data-i18n="navMedia"></a>
  <select id="lang"><option value="pt">🇧🇷 Português</option><option value="en">🇬🇧 English</option></select>
  <span id="status"></span></nav>`);
document.querySelectorAll('nav a').forEach(a => a.classList.toggle('cur', a.pathname === location.pathname || (a.pathname === '/' && location.pathname === '/control.html')));

function applyLang() {
  document.documentElement.lang = lang === 'pt' ? 'pt-BR' : 'en';
  $('lang').value = lang;
  document.querySelectorAll('[data-i18n]').forEach(el => (el.innerHTML = t(el.dataset.i18n)));
  document.querySelectorAll('[data-i18n-ph]').forEach(el => (el.placeholder = t(el.dataset.i18nPh)));
  document.querySelectorAll('[data-i18n-title]').forEach(el => (el.title = t(el.dataset.i18nTitle)));
  if ($('status').dataset.k) $('status').textContent = t($('status').dataset.k);
  if (state) fill();
}
$('lang').onchange = e => { lang = localStorage.lang = e.target.value; applyLang(); };

// Fields with data-k="path.to.value" edit the shared state directly; data-num makes them numeric.
let state, onFill = () => {};
const DEFAULTS = { casters2: '', scene: 'game', camera: 1, windText: 'ja', riichiText: 'ja', startScore: 25000, banners: [], sponsors: [], rotateSecs: 8 };
const get = (o, k) => k.split('.').reduce((a, p) => a[p], o);
const set = (o, k, v) => { const ps = k.split('.'), last = ps.pop(); ps.reduce((a, p) => a[p], o)[last] = v; };

function fill() {
  for (const f of document.querySelectorAll('[data-k]')) if (f !== document.activeElement) f.value = get(state, f.dataset.k);
  onFill();
}

let timer, seq = 0, pending = false; // while our own edit is unsent, ignore echoes so they can't roll it back
function push() {
  fill();
  pending = true;
  const n = ++seq;
  clearTimeout(timer);
  timer = setTimeout(async () => {
    const r = await fetch('/api/state', { method: 'POST', body: JSON.stringify(state) }).catch(() => null);
    if (r?.status === 401) location.href = '/login.html'; // password was reset while this page was open
    if (n === seq) pending = false;
    $('status').textContent = t($('status').dataset.k = r?.ok ? 'saved' : 'offline');
  }, 200);
}

document.addEventListener('input', e => {
  const f = e.target;
  if (!state || !f.dataset.k) return;
  set(state, f.dataset.k, 'num' in f.dataset ? +f.value || 0 : f.value);
  push();
});

// Live sync, so two operators (or a reload) always see the current state.
new EventSource('/api/events').onmessage = e => { if (!pending) { state = { ...DEFAULTS, ...JSON.parse(e.data) }; fill(); } };
