// Shared by control.html, the overlay pages and test.js.
const WINDS = '東南西北';

// Seats are listed in starting order (first seat = East in 東1). Dealer moves one seat per hand number.
const seatWinds = num => [0, 1, 2, 3].map(i => WINDS[(i - (num - 1) % 4 + 4) % 4]);

// 1..4 per seat; ties go to the earlier starting seat, as in standard riichi rules.
function placements(seats) {
  const p = [];
  [0, 1, 2, 3].sort((a, b) => seats[b].score - seats[a].score || a - b).forEach((s, i) => (p[s] = i + 1));
  return p;
}

// "Name 12,5" per line -> {name, total}. Accepts comma or dot decimals.
const parsePlayers = text => text.split('\n').map(l => l.trim()).filter(Boolean).map(l => {
  const m = l.match(/^(.*?)\s+(-?\d+(?:[.,]\d+)?)$/);
  return m ? { name: m[1], total: +m[2].replace(',', '.') } : { name: l, total: 0 };
});
const playersText = players => players.map(p => `${p.name} ${p.total}`).join('\n');

const standings = players => [...players].sort((a, b) => b.total - a.total);

function nextHand(r) {
  const last = r.num >= 4;
  return { ...r, wind: last ? (r.wind + 1) % 4 : r.wind, num: last ? 1 : r.num + 1, honba: 0 };
}

// Declaring riichi pays a 1000 stick to the table; toggling again undoes a misclick.
function toggleRiichi(s, i) {
  const seat = s.seats[i], on = !seat.riichi;
  seat.riichi = on;
  seat.score += on ? -1000 : 1000;
  s.round.riichi = Math.max(0, s.round.riichi + (on ? 1 : -1));
}

const ceil100 = n => Math.ceil(n / 100) * 100;

// Score change per seat for one winner. points = the hand's ron value (e.g. 8000 mangan, 12000 dealer mangan);
// a tsumo splits it the standard way (rounding up to 100 matches the fu/han table, see test.js).
function winDeltas(s, winner, loser, points) {
  const d = [0, 0, 0, 0], honba = s.round.honba, dealer = seatWinds(s.round.num).indexOf('東');
  if (loser != null) {
    d[loser] -= points + 300 * honba;
    d[winner] += points + 300 * honba;
  } else {
    for (let i = 0; i < 4; i++) if (i !== winner) {
      const pay = (winner === dealer ? ceil100(points / 3) : ceil100(points / (i === dealer ? 2 : 4))) + 100 * honba;
      d[i] -= pay;
      d[winner] += pay;
    }
  }
  d[winner] += 1000 * s.round.riichi;
  return d;
}

// Ron, also double/triple: wins = [[seat, points], ...], all paid by loser. Honba and riichi sticks go only to
// the winner closest after the loser in turn order (head bump), as in Mahjong Soul.
function ronDeltas(s, loser, wins) {
  const first = [1, 2, 3].map(k => (loser + k) % 4).find(i => wins.some(w => w[0] === i));
  const none = { round: { ...s.round, honba: 0, riichi: 0 } };
  return wins.map(([w, p]) => winDeltas(w === first ? s : none, w, loser, p))
    .reduce((a, d) => a.map((x, i) => x + d[i]), [0, 0, 0, 0]);
}

// Exhaustive draw: noten seats pay 3000 in total, split among tenpai seats (all or none tenpai: no payment).
function drawDeltas(tenpai) {
  const n = tenpai.filter(Boolean).length;
  return tenpai.map(t => (n % 4 === 0 ? 0 : t ? 3000 / n : -3000 / (4 - n)));
}

// Tiles as mpsz codes ('2m', '7z'), in hand order. Images: public/assets/tiles (FluffyStuff/riichi-mahjong-tiles, public domain).
const TILES = [...'mpsz'].flatMap(suit => [1, 2, 3, 4, 5, 6, 7, 8, 9].slice(0, suit === 'z' ? 7 : 9).map(n => n + suit));
const HONORS = ['Ton', 'Nan', 'Shaa', 'Pei', 'Haku', 'Hatsu', 'Chun'];
const tileSrc = t => `assets/tiles/${t[1] === 'z' ? HONORS[t[0] - 1] : { m: 'Man', p: 'Pin', s: 'Sou' }[t[1]] + t[0]}.svg`;

const fmt = n => Number(n).toLocaleString('pt-BR');
const esc = s => String(s).replace(/[&<>"']/g, c => `&#${c.charCodeAt(0)};`);

// Small Markdown subset for the info screen, read line by line: # headings, **bold**, *italic*, `code`, - / 1. lists,
// - [ ] / - [x] checkboxes, paragraphs (a blank line splits them), line breaks. Escapes first, so typed HTML shows as text.
function md(src) {
  const inline = t => esc(t).replace(/\*\*(.+?)\*\*/g, '<b>$1</b>').replace(/\*(.+?)\*/g, '<i>$1</i>').replace(/`(.+?)`/g, '<code>$1</code>')
    .replace(/^\[([ xX]?)\]\s*/, (_, x) => `<span class="check">${x.trim() ? '☑' : '☐'}</span> `);
  const LISTS = [[/^\s*[-*+]\s+/, 'ul'], [/^\s*\d+[.)]\s+/, 'ol']];
  let out = '', open = null; // open: 'p', 'ul' or 'ol' while a block is running
  const close = () => { if (open) out += open === 'p' ? '</p>' : `</${open}>`; open = null; };
  for (const line of String(src || '').split('\n')) {
    const h = line.match(/^\s*(#{1,3})\s+(.*)/), list = LISTS.find(([re]) => re.test(line));
    if (!line.trim()) close();
    else if (h) { close(); out += `<h${h[1].length}>${inline(h[2])}</h${h[1].length}>`; }
    else if (list) {
      if (open !== list[1]) { close(); out += `<${open = list[1]}>`; }
      out += `<li>${inline(line.replace(list[0], ''))}</li>`;
    } else if (open === 'p') out += `<br>${inline(line.trim())}`;
    else { close(); out += `<p>${inline(line.trim())}`; open = 'p'; }
  }
  close();
  return out;
}

if (typeof module !== 'undefined') module.exports = { seatWinds, placements, parsePlayers, playersText, standings, nextHand, toggleRiichi, winDeltas, ronDeltas, drawDeltas, ceil100, TILES, tileSrc, md };
